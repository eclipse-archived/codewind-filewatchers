# Codewind File Watcher
File watching daemon and clients in Codewind for source code monitoring. 

![platforms](https://img.shields.io/badge/runtime-Java%20%7C%20Node%20%7C%20Go-yellow.svg)
[![License](https://img.shields.io/badge/License-EPL%202.0-red.svg?label=license&logo=eclipse)](https://www.eclipse.org/legal/epl-2.0/)
[![Build Status](https://ci.eclipse.org/codewind/buildStatus/icon?job=Codewind%2Fcodewind-filewatchers%2Fmaster)](https://ci.eclipse.org/codewind/job/Codewind/job/codewind-filewatchers/job/master/)
[![Chat](https://img.shields.io/static/v1.svg?label=chat&message=mattermost&color=145dbf)](https://mattermost.eclipse.org/eclipse/channels/eclipse-codewind)

The intent of this service is to provide mechanism for clients, e.g. IDEs or Che, to notify Codewind to track the list of source file changes and trigger build if auto build has been enabled. There are three implementations on the client written in Java, Typescript and Go to fit different client environments.

## Contributing
We welcome submitting issues and contributions.
1. [Submitting bugs](https://github.com/eclipse/codewind-filewatchers/issues)
2. [Contributing](CONTRIBUTING.md)

## Developing
To develop and debug the Codewind Filewatchers, see [DEVELOPING.md](DEVELOPING.md).

## License
[EPL 2.0](https://www.eclipse.org/legal/epl-2.0/)
