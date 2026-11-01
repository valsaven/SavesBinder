using System;
using System.IO;
using System.Linq;

namespace SavesBinder
{
    /// <summary>
    /// Filesystem primitives for bind/restore/unbind/destroy. All methods run on a worker thread.
    /// </summary>
    internal static class FileOperations
    {
        private static readonly string[] ProtectedPaths =
        {
            @"c:\windows",
            @"c:\program files",
            @"c:\program files (x86)",
            @"c:\programdata",
            @"c:\users\all users",
            @"c:\users\default",
            @"c:\users\public",
            @"c:\$recycle.bin",
            @"c:\system volume information",
        };

        public static string Normalize(string path)
        {
            var full = Path.GetFullPath(path);
            return full.TrimEnd(Path.DirectorySeparatorChar, Path.AltDirectorySeparatorChar);
        }

        /// <summary>Rejects drive roots, protected system directories and paths below a reparse point.</summary>
        public static void ValidatePath(string path)
        {
            var clean = Normalize(path).ToLowerInvariant();
            var root = Path.GetPathRoot(clean)?.TrimEnd(Path.DirectorySeparatorChar);

            if (string.IsNullOrEmpty(clean) || clean == root)
            {
                throw new InvalidOperationException($"Path is a drive root: {path}");
            }

            foreach (var protectedPath in ProtectedPaths)
            {
                if (clean == protectedPath || clean.StartsWith(protectedPath + Path.DirectorySeparatorChar))
                {
                    throw new InvalidOperationException($"Path is a protected system directory: {path}");
                }
            }

            // Check parents only: the path itself may legitimately be a junction (restore/unbind).
            var current = Path.GetDirectoryName(clean);
            while (!string.IsNullOrEmpty(current))
            {
                if ((Directory.Exists(current) || File.Exists(current)) && IsReparsePoint(current))
                {
                    throw new InvalidOperationException($"Path contains a reparse point: {current}");
                }

                var parent = Path.GetDirectoryName(current);
                if (string.Equals(parent, current, StringComparison.OrdinalIgnoreCase))
                {
                    break;
                }

                current = parent;
            }
        }

        public static bool IsReparsePoint(string path)
        {
            return (File.GetAttributes(path) & FileAttributes.ReparsePoint) != 0;
        }

        /// <summary>Attributes, or null when the path does not exist (a dangling junction still reports attributes).</summary>
        public static FileAttributes? GetAttributesOrNull(string path)
        {
            try
            {
                return File.GetAttributes(path);
            }
            catch (FileNotFoundException)
            {
                return null;
            }
            catch (DirectoryNotFoundException)
            {
                return null;
            }
            catch (ArgumentException)
            {
                return null;
            }
        }

        /// <summary>
        /// True only when the path can actually be opened as a directory. Directory.Exists stays true
        /// for a dangling junction, while enumeration fails.
        /// </summary>
        public static bool IsReachableDirectory(string path)
        {
            try
            {
                Directory.GetFileSystemEntries(path);
                return true;
            }
            catch (DirectoryNotFoundException)
            {
                return false;
            }
            catch (IOException)
            {
                return false;
            }
            catch (ArgumentException)
            {
                return false;
            }
            catch (UnauthorizedAccessException)
            {
                // Exists but is not readable; treat it as healthy rather than recreating the link.
                return true;
            }
        }

        public static bool IsNestedPath(string first, string second)
        {
            var a = Normalize(first).ToLowerInvariant();
            var b = Normalize(second).ToLowerInvariant();
            if (a == b)
            {
                return true;
            }

            var separator = Path.DirectorySeparatorChar.ToString();
            return (a + separator).StartsWith(b + separator, StringComparison.Ordinal)
                || (b + separator).StartsWith(a + separator, StringComparison.Ordinal);
        }

        public static void CheckWritePermission(string path)
        {
            var directory = Directory.Exists(path) ? path : Path.GetDirectoryName(Path.GetFullPath(path));
            if (string.IsNullOrEmpty(directory) || !Directory.Exists(directory))
            {
                throw new DirectoryNotFoundException($"Directory not found: {directory}");
            }

            var probe = Path.Combine(directory, $".write_test_{Guid.NewGuid():N}.tmp");
            try
            {
                using (File.Create(probe))
                {
                }
            }
            finally
            {
                if (File.Exists(probe))
                {
                    File.Delete(probe);
                }
            }
        }

        public static void CheckDiskSpace(string source, string destination)
        {
            var (_, size) = CountFilesAndSize(source);
            if (size == 0)
            {
                return;
            }

            var root = Path.GetPathRoot(Path.GetFullPath(destination));
            if (string.IsNullOrEmpty(root))
            {
                return;
            }

            long free;
            try
            {
                free = new DriveInfo(root).AvailableFreeSpace;
            }
            catch (ArgumentException)
            {
                return;
            }
            catch (IOException)
            {
                return;
            }

            if (free < size)
            {
                throw new InvalidOperationException($"Insufficient disk space: need {size} bytes, available {free} bytes.");
            }
        }

        public static (int Count, long Size) CountFilesAndSize(string root)
        {
            var count = 0;
            long size = 0;

            foreach (var file in Directory.EnumerateFiles(root, "*", SearchOption.AllDirectories))
            {
                count++;
                size += new FileInfo(file).Length;
            }

            return (count, size);
        }

        public static string GenerateUniqueName(string path)
        {
            if (!File.Exists(path) && !Directory.Exists(path))
            {
                return path;
            }

            var directory = Path.GetDirectoryName(path) ?? string.Empty;
            var name = Path.GetFileNameWithoutExtension(path);
            var extension = Path.GetExtension(path);

            for (var i = 1; ; i++)
            {
                var candidate = Path.Combine(directory, $"{name} ({i}){extension}");
                if (!File.Exists(candidate) && !Directory.Exists(candidate))
                {
                    return candidate;
                }
            }
        }

        /// <summary>
        /// Copies directory contents with backup and integrity verification, then removes the source.
        /// Returns the backup path; the caller deletes it after the operation fully succeeds.
        /// </summary>
        public static string MoveDirectorySafe(string source, string destination, string backupRoot)
        {
            var backupPath = BackupDirectory(source, backupRoot);

            // Source is untouched until the copy is verified, so a failure here only needs the partial copy removed.
            try
            {
                CopyDirectory(source, destination, resolveConflicts: true);
                VerifyMoveIntegrity(source, destination);
            }
            catch
            {
                TryDeleteDirectory(destination);
                TryDeleteDirectory(backupPath);
                throw;
            }

            try
            {
                Directory.Delete(source, recursive: true);
            }
            catch
            {
                // Source is partially removed: restore it from the backup so no data is lost.
                TryDeleteDirectory(destination);
                try
                {
                    TryDeleteDirectory(source);
                    CopyDirectory(backupPath, source, resolveConflicts: false);
                    TryDeleteDirectory(backupPath);
                }
                catch
                {
                    // Keep the backup for manual recovery.
                }

                throw;
            }

            return backupPath;
        }

        public static void TryDeleteDirectory(string path)
        {
            try
            {
                if (Directory.Exists(path))
                {
                    Directory.Delete(path, recursive: true);
                }
            }
            catch (IOException)
            {
            }
            catch (UnauthorizedAccessException)
            {
            }
        }

        /// <summary>
        /// Removes a link without following it. A real non-empty directory is never deleted.
        /// </summary>
        public static void RemoveLink(string path)
        {
            var attributes = GetAttributesOrNull(path);
            if (attributes is null)
            {
                return;
            }

            if ((attributes.Value & FileAttributes.ReparsePoint) != 0)
            {
                if ((attributes.Value & FileAttributes.Directory) != 0)
                {
                    Directory.Delete(path);
                }
                else
                {
                    File.Delete(path);
                }

                return;
            }

            if ((attributes.Value & FileAttributes.Directory) != 0 && !Directory.EnumerateFileSystemEntries(path).Any())
            {
                Directory.Delete(path);
                return;
            }

            throw new InvalidOperationException($"Refusing to delete a real non-empty path: {path}");
        }

        private static string BackupDirectory(string source, string backupRoot)
        {
            Directory.CreateDirectory(backupRoot);

            var name = Path.GetFileName(Normalize(source));
            var backupPath = Path.Combine(backupRoot, $"{name}_{DateTime.Now:yyyyMMdd_HHmmss_fff}_{Guid.NewGuid():N}");
            try
            {
                CopyDirectory(source, backupPath, resolveConflicts: false);
            }
            catch
            {
                TryDeleteDirectory(backupPath);
                throw;
            }

            return backupPath;
        }

        private static void CopyDirectory(string source, string destination, bool resolveConflicts)
        {
            Directory.CreateDirectory(destination);

            foreach (var entry in new DirectoryInfo(source).EnumerateFileSystemInfos())
            {
                var targetPath = Path.Combine(destination, entry.Name);
                if (resolveConflicts && (File.Exists(targetPath) || Directory.Exists(targetPath)))
                {
                    targetPath = GenerateUniqueName(targetPath);
                }

                if (entry is DirectoryInfo directory)
                {
                    CopyDirectory(directory.FullName, targetPath, resolveConflicts);
                }
                else
                {
                    File.Copy(entry.FullName, targetPath, overwrite: false);
                }
            }
        }

        private static void VerifyMoveIntegrity(string source, string destination)
        {
            var (sourceCount, sourceSize) = CountFilesAndSize(source);
            var (destinationCount, destinationSize) = CountFilesAndSize(destination);

            if (sourceCount != destinationCount)
            {
                throw new IOException($"File count mismatch: source={sourceCount}, destination={destinationCount}.");
            }

            if (sourceSize != destinationSize)
            {
                throw new IOException($"Size mismatch: source={sourceSize}, destination={destinationSize}.");
            }
        }
    }
}
