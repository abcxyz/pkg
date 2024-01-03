## ReflectionClient Usage

To use the reflectionclient in your codebase you'll want to set up the client
with some arguments:

```
client, err := reflectionclient.NewClient(
    ctx,
    &reflectionclient.ClientConfig{
        Host:     fmt.Sprintf("%s:%d", serverAddress, serverPort),
        Audience: "https://" + serverAddress,
        Insecure: false,
        Timeout:  10*time.Second,
    },
)
```

then make calls in your code like so:

```
response, err := client.CallMethod(ctx, "google.com.exampleservice.v1.Foo.Bar", "", false)
```
