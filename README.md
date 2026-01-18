# grpcexp

`grpcexp` is an interactive explorer for interacting with grpc servers. It's basically a tui on top of [`grpcurl`](https://github.com/fullstorydev/grpcurl).

![Demo](demo.svg)

## Installation

### Linux or MacOS

You can install the latest version of `grpcexp` by running the following command in your terminal.

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/prnvbn/grpcexp/main/installer.sh)"
```

Move the binary to a directory in your PATH. For e.g. `/usr/local/bin` on linux.

### via `homebrew`

You can install `grpcexp` using the (prnvbn/homebrew-tap)[https://github.com/prnvbn/homebrew-tap].

```bash
brew install prnvbn/tap/grpcexp
```

### via `go install`

```bash
go install github.com/prnvbn/grpcexp/cmd/grpcexp@latest
```

### Windows

Windows installation instructions are a WIP. In the meantime, you can download the latest release from the [releases page](https://github.com/prnvbn/grpcexp/releases)

> [!NOTE]
>  
> To update `grpcexp` to the latest version, simply re-run any of the installation methods above.
> They always install the most recent release.

#### Enabling Command Autocompletion

To enable autocomplete, add the following to your `.bashrc` or `.bash_profile` file:

```bash
# you can also generate completions for zsh and fish shells by replacing bash with zsh or fish
source <(grpcexp completion bash)
```

## Why

Let me preface this by saying, I really like grpcurl but have a few nits:

<details>

<summary>I have to run ~5 commands to make one grpc call</summary>

```shell
$ grpcurl -plaintext :50051 list
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
helloworld.Greeter
```

```shell
$ grpcurl -plaintext :50051 list helloworld.Greeter
helloworld.Greeter.SayHello
```

```shell
$ grpcurl -plaintext :50051 describe helloworld.Greeter.SayHello
helloworld.Greeter.SayHello is a method:
rpc SayHello ( .helloworld.HelloRequest ) returns ( .helloworld.HelloReply );
```

```shell
$ grpcurl -plaintext :50051 describe helloworld.HelloRequest
helloworld.HelloRequest is a message:
message HelloRequest {
string name = 1;
}
```

```shell
$ grpcurl -plaintext -d '{"name": "joe"}' :50051 helloworld.Greeter.SayHello
{
"message": "Hello joe"
}
```

</details>

<details>

<summary>the lack of POSIX/GNU-style --flags</summary>

personal taste.

</details>

<details>

<summary>manual JSON construction for complex types</summary>

With `grpcurl`, you have to manually construct JSON for nested messages, maps, lists, and oneofs which is very tedious.

</details>

## Contributing

Feel free to open an issue or submit a pull request. I'm always open to suggestions and improvements :)
