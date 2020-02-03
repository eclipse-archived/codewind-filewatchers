
# Developing the Codewind Filewatchers

## Building and testing the Java filewatcher

The Java filewatcher daemon is fully integrated into the [Codewind Eclipse](https://github.com/eclipse/codewind-eclipse#) codebase. To build and run Codewind Eclipse with this filewatcher, you only need to follow [the standard 'Developing Codewind Eclipse' instructions](https://github.com/eclipse/codewind-eclipse#developing-codewind-for-eclipse).


## Building and testing the Node filewatcher

If running on Windows, run `npm-package.sh` using [WSL](https://docs.microsoft.com/en-us/windows/wsl/about) or [MSYS2](https://www.msys2.org/).

#### A) Build Filewatcher package
```
git clone https://github.com/codewind-eclipse/codewind-filewatchers
cd codewind-filewatchers
cd Filewatcherd-TypeScript
./npm-package.sh
```
This will produce a `filewatcherd-node_(version).tar.gz` file.


#### B) Setup a VS Code Development environment

Follow the [Developing Codewind for VS Code](https://github.com/eclipse/codewind-vscode/blob/master/DEVELOPING.md) to setup a VS Code development environment that contains the Codewind VS Code extension source.

#### C) Install the new `filewatcher.tar.gz` into VS Code
```
cd (path to git repository from step B)/codewind-vscode/dev
npm uninstall codewind-filewatcher
npm install (path to `filewatcherd-(version).tar.gz` from step A)
```

#### D) Launch VS Code

Restart VS Code. Then, hit F5 to launch the debugger. It should build, compile, and start the Codewind VS Code extension. See [DEVELOPING.md](https://github.com/eclipse/codewind-vscode/blob/master/DEVELOPING.md) for additional information on launching a VS Code extension.


# How to view the Codewind Filewatchers logs

The filewatcher daemons automatically log filewatcher-specific log statements to a codewind-specific directory on the file system. At most the last 24MB of log files will be retained (rolling logs), and any existing log files are deleted on IDE startup.


### Locations of filewatcher log file(s), on the file system:

- When running in Codewind Eclipse: `(eclipse workspace)/.metadata`
- When running in Codewind VSCode:
  - In VSCode, `Help` > `Toggle Developer Tools` > `Console`
  - Then in the Filter box, enter `Logger initialized at`. For example: `Logger.js.setLogFilePath():62]: Logger initialized at c:\Users\JONATHANWest\AppData\Roaming\Code\logs\20190624T145813\exthost3\IBM.codewind\codewind.log`
  - Open the directory containing `codewind.log`, and look for `filewatcherd-(...).log`
- When running standalone: `(user home dir)/.codewind`


### Log filename, in above directory:  
- `filewatcherd-(#).log` (there should exist at least one, and at most two, log files; the log file with the larger `#` is the most recent.


### How to switch a filewatcher to the more verbose 'DEBUG' log level

Filewatchers default to an `INFO` log level. If you wish to see (or are asked to enable) the more verbose `DEBUG` log level, do as follows.

#### Steps:
1. Close VSCode and/or Eclipse, to ensure the existing filewatcher is stopped.
2. Set the following [system environment variable](https://superuser.com/questions/284342/what-are-path-and-other-environment-variables-and-how-can-i-set-or-use-them):
- `filewatcher_log_level` to `debug`
3. Open a new Command Prompt or Terminal and type:
- Windows: `echo %filewatcher_log_level%`
- Linux/MacOS: `echo $filewatcher_log_level`
- In both cases, you should see `debug`. If you do not, your system environment variable is not set correctly.
4. Start VSCode or Eclispe.

To verify the change has taken effect, open the `filewatcherd-(...).log` file and look for a line like this:
```
codewind-filewatcher logging to C:\Users\your-username\.codewind with log level DEBUG
```


