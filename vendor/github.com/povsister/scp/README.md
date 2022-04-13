# Secure Copy Protocol implemented in Go

[![GoReport Widget]][GoReport]
[![GoPkg Widget]][GoPkg]

## Overview
Production-ready Secure Copy Protocol (aka: SCP) implemented in Go with 
well documentation and neat dependency. 

## Introduction
[Secure Copy Protocol][SCP Wiki] uses Secure Shell (SSH) to 
transfer files between host on a network.

There is no RFC that defines the specifics of the protocol.
This package simply implements SCP against the [OpenSSH][OpenSSH]'s `scp` tool, 
thus you can directly transfer files to/from *uinx system within your Go code, 
as long as the remote host has OpenSSH installed.

## Features
* Copy file from local to remote.
* Copy file from remote to local.
* Copy from buffer to remote file. (e.g: copy from `bytes.Reader`)
* Copy from remote file to buffer. (e.g: copy to `os.Stdout`)
* Recursively copy directory from local to remote.
* Recursively copy directory from remote to local.
* Set permission bits for transferred files.
* Set timeout/context for transfer.
* Preserve the permission bits and modification time at transfer.
* No resources leak. (e.g: goroutine, file descriptor)
* Low memory consuming for transferring huge files.
* TODO:
  * Transfer speed limit.
  * Performance benchmark/optimization for lots of small files.
* Won't support:
  * Copy file from remote to remote.

## Install
```go
go get github.com/povsister/scp
```

## Example usage

This package leverages `golang.org/x/crypto/ssh` to establish a SSH connection to remote host.

*Error handling are omitted in examples!*

### Copy a file to remote
```go
// Build a SSH config from username/password
sshConf := scp.NewSSHConfigFromPassword("username", "password")

// Build a SSH config from private key
privPEM, err := ioutil.ReadFile("/path/to/privateKey")
// without passphrase
sshConf, err := scp.NewSSHConfigFromPrivateKey("username", privPEM)
// with passphrase
sshConf, err := scp.NewSSHConfigFromPrivateKey("username", privPEM, passphrase)

// Dial SSH to "my.server.com:22".
// If your SSH server does not listen on 22, simply suffix the address with port.
// e.g: "my.server.com:1234"
scpClient, err := scp.NewClient("my.server.com", sshConf, &scp.ClientOption{})

// Build a SCP client based on existing "golang.org/x/crypto/ssh.Client"
scpClient, err := scp.NewClientFromExistingSSH(existingSSHClient, &scp.ClientOption{})

defer scpClient.Close()


// Do the file transfer without timeout/context
err = scpClient.CopyFileToRemote("/path/to/local/file", "/path/at/remote", &scp.FileTransferOption{})

// Do the file copy with timeout, context and file properties preserved.
// Note that the context and timeout will both take effect.
fo := &scp.FileTransferOption{
    Context: yourCotext,
    Timeout: 30 * time.Second, 
    PreserveProp: true,
}
err = scpClient.CopyFileToRemote("/path/to/local/file", "/path/at/remote", fo)
```

### Copy a file from remote
```go
// Copy the file from remote and save it as "/path/to/local/file".
err = scpClient.CopyFileFromRemote("/path/to/remote/file", "/path/to/local/file", &scp.FileTransferOption{})

// Copy the remote file and print it in Stdout.
err = scpClient.CopyFromRemote("/path/to/remote/file", os.Stdout, &scp.FileTransferOption{})
```

### Copy from buffer to remote as a file
```go
// From buffer
buffer := []byte("something excited")
reader := bytes.NewReader(buffer)

// From fd
// Note that its YOUR responsibility to CLOSE the fd after transfer.
reader, err := os.Open("/path/to/local/file")
defer reader.Close()


// Note that the reader must implement "KnownSize" interface except os.File
// For the content length must be provided before transfer.
// The last part of remote location will be used as file name at remote.
err := scpClient.CopyToRemote(reader, "/path/to/remote/file", &scp.FileTransferOption{})
```

### Recursively copy a directory to remote
```go
// recursively copy to remote
err := scpClient.CopyDirToRemote("/path/to/local/dir", "/path/to/remote/dir", &scp.DirTransferOption{})

// recursively copy to remote with timeout, context and file properties.
// Note that the context and timeout will both take effect.
do := &scp.DirTransferOption{
    Context: yourContext,
    Timeout: 10 * time.Minute,
    PreserveProp: true,
}
err:= scpClient.CopyDirToRemote("/path/to/local/dir", "/path/to/remote/dir", do)
```

### Recursively copy a directory from remote
```go
// recursively copy from remote.
// The content of remote dir will be save under "/path/to/local".
err := scpClient.CopyDirFromRemote("/path/to/remote/dir", "/path/to/local", &scp.DirTransferOption{})
```

## Something you need to know
`SCP` is a light-weighted protocol which implements file transfer only. It does not support 
advanced features like: directory listing, resume from break-point.

So, it's commonly used for transferring some small-size, temporary files. If you heavily 
depend on the file transfer, you may consider using `SFTP` instead.

Another thing you may notice is that I didn't put `context.Context` as the first argument in
function signature. Instead, it's located in `TransferOption`. This is intentional because it
makes the API also light-weighted.

## License
[MIT License][MIT License]

[MIT License]: https://en.wikipedia.org/wiki/MIT_License
[OpenSSH]: https://www.openssh.com
[SCP Wiki]: https://en.wikipedia.org/wiki/Secure_copy_protocol
[GoPkg]: https://pkg.go.dev/github.com/povsister/scp
[GoPkg Widget]: https://pkg.go.dev/badge/github.com/povsister/scp.svg
[GoReport]: https://goreportcard.com/report/povsister/scp
[GoReport Widget]: https://goreportcard.com/badge/povsister/scp
