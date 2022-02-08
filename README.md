# Waarp Gateway

*Waarp-Gateway* is an MFT solution which allow making monitored file 
transfers with various protocols. Those protocols are all interchangeable, 
meaning the gateway is typically used for protocol interoperability and 
breaking.

## Features

- Monitoring & traceability for all transfers
- Ability to execute pre & post tasks
- Make transfers both as a server and as a client
- Supports multiple databases: SQLite (embedded), PostgreSQL & 
MySQL (+its variants)
- Supports multiple protocols:
  - R66 / R66-TLS
  - SFTP
  - HTTP / HTTPS
- Works in clusters with a load balancer
- Administration via a REST API & command line interface

##Getting started

### Build from source

*Waarp-Gateway* requires Go version 1.17 or later to compile. Since
the gateway also calls some C code, GCC (or mingw on Windows) is 
also required on the machine to compile the program.

```shell
git clone https://code.waarp.fr/apps/gateway/gateway
./make.sh build
```

The binaries will be written in the ``build`` directory under the
project's root directory. Note that this will only build the service
and command line binaries, and only for the local machine's OS and 
architecture.

### Run the tests

To run the classic test suite, run the following command:

```shell
./make.sh test
```

By default, this will run all the tests using SQLite as a test
database. To run the tests with other types of database, set the 
``GATEWAY_TEST_DB`` environment variable to either ``postgresql`` 
or ``mysql`` and run the command again. This requires a test database
to be preconfigured on the local machine.

For PostgreSQL, the test database must be named ``waarp_gateway_test``,
and the server must be running on the default port (5432) with the
default user enabled (user ``postgres`` with no password).

For MySQL, the database must also be named ``waarp_gateway_test``, and
the default MySQL user enabled (user ``root`` with no password) on the
default port (3306).

### Run the linter

*Waarp-Gateway* uses the golangci-lint linter to check the code formatting
and to check some for some basic coding errors. To run the linter, run
the command:

```shell
./make check
```

### Build the documentation

***Note:*** *Currently, the *Waarp-Gateway* documentation is only available
in French.*

The *Waarp-Gateway* documentation is written in rst format, and built using
the Sphinx documentation generator. As such, Python 3 and virtualenv are
both required to build the documentation. Once these requirements, are
satisfied, run the following command to build the documentation:

```shell
./make.sh doc
```

The documentation will be written in HTML under the ``doc/build`` 
directory. Alternatively, you can use the command :

```shell
./make.sh doc watch
```

which will build the documentation, and then start a local HTTP server
on port 8082 with the documentation hosted on it.

## Support

[Waarp](https://waarp.fr) provides professional support and services for
*Waarp-Gateway*.

## Licence

