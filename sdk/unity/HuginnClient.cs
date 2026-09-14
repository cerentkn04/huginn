using System;
using System.Collections;
using System.IO;
using UnityEngine;
using UnityEngine.Networking;

public struct HuginnServer
{
    public string Address;
    public string JoinCode;
    public int PlayerCount;
    public int MaxPlayers;
}

public class HuginnClient : MonoBehaviour
{
    [SerializeField] private string defaultBaseUrl = "http://localhost:8080";

    public static HuginnClient Instance { get; private set; }
    private string baseUrl;

    void Awake()
    {
        if (Instance != null && Instance != this) { Destroy(gameObject); return; }
        Instance = this;
        baseUrl = ResolveBaseUrl(defaultBaseUrl);
    }

    public void FindServer(Action<HuginnServer> onSuccess, Action<string> onError)
        => StartCoroutine(Request("/api/servers/available", onSuccess, onError));

    public void FindServerByCode(string code, Action<HuginnServer> onSuccess, Action<string> onError)
        => StartCoroutine(Request($"/api/servers/code/{code}", onSuccess, onError));

    IEnumerator Request(string path, Action<HuginnServer> onSuccess, Action<string> onError)
    {
        using var req = UnityWebRequest.Get(baseUrl + path);
        yield return req.SendWebRequest();

        if (req.result != UnityWebRequest.Result.Success)
        {
            onError?.Invoke(req.error);
            yield break;
        }

        var inst = JsonUtility.FromJson<InstanceResponse>(req.downloadHandler.text);
        if (inst == null || string.IsNullOrEmpty(inst.Address))
        {
            onError?.Invoke("invalid response from Huginn");
            yield break;
        }

        onSuccess?.Invoke(new HuginnServer
        {
            Address = inst.Address,
            JoinCode = inst.JoinCode,
            PlayerCount = inst.PlayerCount,
            MaxPlayers = inst.MaxPlayers,
        });
    }

    static string ResolveBaseUrl(string fallback)
    {
        var args = Environment.GetCommandLineArgs();
        for (int i = 0; i < args.Length - 1; i++)
        {
            if (args[i] == "-huginn")
            {
                Debug.Log($"[HUGINN] base url from command line: {args[i + 1]}");
                return args[i + 1];
            }
        }

        string path = Path.Combine(Application.dataPath, "..", "huginn.json");
        if (File.Exists(path))
        {
            var cfg = JsonUtility.FromJson<ClientConfig>(File.ReadAllText(path));
            if (cfg != null && !string.IsNullOrEmpty(cfg.huginnBaseUrl))
            {
                Debug.Log($"[HUGINN] base url from {path}: {cfg.huginnBaseUrl}");
                return cfg.huginnBaseUrl;
            }
        }

        Debug.Log($"[HUGINN] base url falling back to inspector value: {fallback}");
        return fallback;
    }

    [Serializable] class ClientConfig { public string huginnBaseUrl; }

    [Serializable]
    class InstanceResponse
    {
        public string Address;
        public string JoinCode;
        public int PlayerCount;
        public int MaxPlayers;
        public string State;
    }
}