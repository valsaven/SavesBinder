using System.Collections.Generic;
using System.IO;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace SavesBinder
{
    /// <summary>
    /// JSON database file: { "links": [ { "target": "...", "link": "..." } ] }
    /// target = stored saves path, link = junction at the original location.
    /// </summary>
    public class LinkDatabase
    {
        private static readonly JsonSerializerOptions ReadOptions = new()
        {
            PropertyNameCaseInsensitive = true,
        };

        private static readonly JsonSerializerOptions WriteOptions = new()
        {
            WriteIndented = true,
        };

        [JsonPropertyName("links")]
        public List<SaveItem> Links { get; set; } = new();

        public static LinkDatabase Load(string path)
        {
            using var stream = File.OpenRead(path);
            var db = JsonSerializer.Deserialize<LinkDatabase>(stream, ReadOptions) ?? new LinkDatabase();
            db.Links ??= new List<SaveItem>();
            db.Links.RemoveAll(item => item is null); // hand-edited JSON can contain null array entries
            return db;
        }

        /// <summary>Writes via a temp file and rename so a crash cannot truncate the database.</summary>
        public void Save(string path)
        {
            var fullPath = Path.GetFullPath(path);
            var directory = Path.GetDirectoryName(fullPath);
            if (!string.IsNullOrEmpty(directory))
            {
                Directory.CreateDirectory(directory);
            }

            var tmpPath = fullPath + ".tmp";
            try
            {
                File.WriteAllText(tmpPath, JsonSerializer.Serialize(this, WriteOptions));
                File.Move(tmpPath, fullPath, overwrite: true);
            }
            finally
            {
                if (File.Exists(tmpPath))
                {
                    File.Delete(tmpPath);
                }
            }
        }
    }

    public class SaveItem
    {
        [JsonPropertyName("target")]
        public string Target { get; set; } = string.Empty;

        [JsonPropertyName("link")]
        public string Link { get; set; } = string.Empty;

        [JsonIgnore]
        public string TargetText => $"target: \"{Target}\"";

        [JsonIgnore]
        public string LinkText => $"link: \"{Link}\"";
    }
}
