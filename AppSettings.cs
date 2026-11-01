namespace SavesBinder
{
    public class AppSettings
    {
        public string DbFilePath { get; set; } = "";
        public string StoragePath { get; set; } = "";
        public string SavesPath { get; set; } = "";
        public string Language { get; set; } = "ru";
        public int WindowX { get; set; }
        public int WindowY { get; set; }
        public int WindowWidth { get; set; }
        public int WindowHeight { get; set; }
        public bool IsMaximized { get; set; }
        public bool ConfirmActions { get; set; } = true;
        public bool CenterOnStartup { get; set; }
    }
}