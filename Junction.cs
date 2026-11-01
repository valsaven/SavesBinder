using System;
using System.Buffers.Binary;
using System.ComponentModel;
using System.IO;
using System.Runtime.InteropServices;
using System.Text;
using Microsoft.Win32.SafeHandles;

namespace SavesBinder
{
    /// <summary>
    /// NTFS directory junctions. Junctions do not require administrator rights,
    /// unlike symbolic links, and are what the original Go app created.
    /// </summary>
    internal static class Junction
    {
        private const uint GenericWrite = 0x40000000;
        private const uint OpenExisting = 3;
        private const uint FileFlagOpenReparsePoint = 0x00200000;
        private const uint FileFlagBackupSemantics = 0x02000000;
        private const uint FsctlSetReparsePoint = 0x000900A4;
        private const uint IOReparseTagMountPoint = 0xA0000003;

        [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
        private static extern SafeFileHandle CreateFileW(
            string fileName,
            uint desiredAccess,
            uint shareMode,
            IntPtr securityAttributes,
            uint creationDisposition,
            uint flagsAndAttributes,
            IntPtr templateFile);

        [DllImport("kernel32.dll", SetLastError = true)]
        private static extern bool DeviceIoControl(
            SafeFileHandle device,
            uint ioControlCode,
            byte[] inBuffer,
            uint inBufferSize,
            IntPtr outBuffer,
            uint outBufferSize,
            out uint bytesReturned,
            IntPtr overlapped);

        public static void Create(string linkPath, string targetPath)
        {
            var target = FileOperations.Normalize(targetPath);
            var link = FileOperations.Normalize(linkPath);

            if (!Directory.Exists(target))
            {
                throw new DirectoryNotFoundException($"Junction target does not exist: {target}");
            }

            if (Directory.Exists(link) || File.Exists(link))
            {
                throw new IOException($"Link path already exists: {link}");
            }

            var parent = Path.GetDirectoryName(link);
            if (string.IsNullOrEmpty(parent) || !Directory.Exists(parent))
            {
                throw new DirectoryNotFoundException($"Link parent directory does not exist: {parent}");
            }

            Directory.CreateDirectory(link);

            try
            {
                using var handle = CreateFileW(
                    link,
                    GenericWrite,
                    0,
                    IntPtr.Zero,
                    OpenExisting,
                    FileFlagOpenReparsePoint | FileFlagBackupSemantics,
                    IntPtr.Zero);

                if (handle.IsInvalid)
                {
                    throw new Win32Exception(Marshal.GetLastWin32Error(), "Cannot open link path.");
                }

                var substituteName = Encoding.Unicode.GetBytes(@"\??\" + target + "\0");
                var printName = Encoding.Unicode.GetBytes(target + "\0");
                var reparseDataLength = 8 + substituteName.Length + printName.Length;
                var buffer = new byte[8 + reparseDataLength];

                BinaryPrimitives.WriteUInt32LittleEndian(buffer.AsSpan(0, 4), IOReparseTagMountPoint);
                BinaryPrimitives.WriteUInt16LittleEndian(buffer.AsSpan(4, 2), (ushort)reparseDataLength);
                BinaryPrimitives.WriteUInt16LittleEndian(buffer.AsSpan(8, 2), 0);
                // Lengths exclude the terminating NUL; the print name starts right after the substitute NUL.
                BinaryPrimitives.WriteUInt16LittleEndian(buffer.AsSpan(10, 2), (ushort)(substituteName.Length - 2));
                BinaryPrimitives.WriteUInt16LittleEndian(buffer.AsSpan(12, 2), (ushort)substituteName.Length);
                BinaryPrimitives.WriteUInt16LittleEndian(buffer.AsSpan(14, 2), (ushort)(printName.Length - 2));

                Buffer.BlockCopy(substituteName, 0, buffer, 16, substituteName.Length);
                Buffer.BlockCopy(printName, 0, buffer, 16 + substituteName.Length, printName.Length);

                if (!DeviceIoControl(handle, FsctlSetReparsePoint, buffer, (uint)buffer.Length,
                        IntPtr.Zero, 0, out _, IntPtr.Zero))
                {
                    throw new Win32Exception(Marshal.GetLastWin32Error(), "Cannot set junction reparse point.");
                }
            }
            catch
            {
                try
                {
                    if (Directory.Exists(link))
                    {
                        // Non-recursive: a junction must not be followed, and the directory should be empty.
                        Directory.Delete(link);
                    }
                }
                catch
                {
                    // Leave the empty directory for the next run.
                }

                throw;
            }
        }
    }
}
