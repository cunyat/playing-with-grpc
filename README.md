This is only a proof of concept about handling gRPC requests dynamically. 
For example for building gRPC mocks, gateways...

To test it, you need to generate a description file for your proto file,
and pass the generated filename as argument:

```shell
protoc --descriptor_set_out=player.pb player.proto
go run main.go player.pb
```
