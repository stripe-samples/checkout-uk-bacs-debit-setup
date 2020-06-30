using Newtonsoft.Json;

public class ConfigResponse
{
    [JsonProperty("publicKey")]
    public string PublishableKey { get; set; }
}