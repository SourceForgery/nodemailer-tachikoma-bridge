## Dependencies

```bash
sudo apt install protobuf-compiler
```

 https://github.com/subodh-dharma/go-generate-protoc

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```

Build and push container
```
docker build -t sourceforgery/nodemailer-tachikoma-bridge .
docker push sourceforgery/nodemailer-tachikoma-bridge
```