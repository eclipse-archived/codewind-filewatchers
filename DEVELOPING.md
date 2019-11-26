
# Developing the Codewind Filewatchers


## Building and testing the Java filewatcher

The Java filewatcher daemon is fully integrated into the [Codewind Eclipse](https://github.com/eclipse/codewind-eclipse#) codebase. To build and run Codewind Eclipse with this filewatcher, you only need to follow [the standard 'Developing Codewind Eclipse' instructions](https://github.com/eclipse/codewind-eclipse#developing-codewind-for-eclipse).


## Building and testing the Node filewatcher

If running on Windows, run `npm-package.sh` using [WSL](https://docs.microsoft.com/en-us/windows/wsl/about) or [MSYS2](https://www.msys2.org/).

(Run on Mac or Linux, don't use Windows)

#### A) Build Filewatcher package
```
git clone https://github.com/codewind-eclipse/codewind-filewatchers
cd codewind-filewatchers
cd Filewatcherd-TypeScript
./npm-package.sh
```
This will produce a `filewatcherd-node_(version).tar.gz` file.


#### B) Setup a VSCode Development environment

Follow the [Developing VSCode instructions](https://github.com/eclipse/codewind-vscode/blob/master/DEVELOPING.md) to setup a VSCode development environment that contains the Codewind VSCode extension source.

#### C) Install the new filewatcher.tar.gz into VSCode
```
cd (path to git repository from step B)/codewind-vscode/dev
npm uninstall codewind-filewatcher
npm install (path to `filewatcherd-(version).tar.gz` from step A)
```

#### D) Launch VSCode

Restart VSCode. Then, hit F5 to launch the debugger. It should build, compile, and start the Codewind VSCode extension. See [DEVELOPING.md](https://github.com/eclipse/codewind-vscode/blob/master/DEVELOPING.md) for additional information on launching a VSCode extension.



