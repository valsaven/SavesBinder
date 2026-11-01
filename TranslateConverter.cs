using Microsoft.UI.Xaml;
using Microsoft.UI.Xaml.Data;
using System;

namespace SavesBinder
{
    public class TranslateConverter : IValueConverter
    {
        public object Convert(object value, Type targetType, object parameter, string language)
        {
            // The parameter is the translation key passed from XAML.
            if (parameter is string key)
            {
                // Get our LangManager from the application resources.
                if (Application.Current.Resources.TryGetValue("Loc", out var locObj) && locObj is LangManager loc)
                {
                    return loc.Get(key);
                }
            }
            return string.Empty;
        }

        public object ConvertBack(object value, Type targetType, object parameter, string language)
        {
            throw new NotImplementedException();
        }
    }
}