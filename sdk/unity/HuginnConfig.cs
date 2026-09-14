using System;
using System.IO;
using UnityEngine;

[Serializable]
class HuginnClientConfig
{
    public string huginnBaseUrl;
}

public static class HuginnConfig
{
    public static string Load(string fallback)
    {
        var args = Environment.GetCommandLineArgs();
        for (int i = 0; i < args.Length - 1; i++)
        {
            if (args[i] == "-huginn")
            {
                Debug.Log($"[CONFIG] huginn url from command line: {args[i + 1]}");
                return args[i + 1];
            }
        }

        string path = Path.Combine(Application.dataPath, "..", "huginn.json");
        if (File.Exists(path))
        {
            try
            {
                var cfg = JsonUtility.FromJson<HuginnClientConfig>(File.ReadAllText(path));
                if (!string.IsNullOrEmpty(cfg.huginnBaseUrl))
                {
                    Debug.Log($"[CONFIG] huginn url from {path}: {cfg.huginnBaseUrl}");
                    return cfg.huginnBaseUrl;
                }
            }
            catch (Exception e)
            {
                Debug.LogError($"[CONFIG] failed to read {path}: {e.Message}");
            }
        }

        Debug.Log($"[CONFIG] huginn url falling back to inspector value: {fallback}");
        return fallback;
    }
}