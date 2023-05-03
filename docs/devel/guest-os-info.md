## Guest Operating System Information

Guest operating system identity for the Virtual Machine will be provided by the label ``kubevirt.io/os`` :

```
metadata:
  name: myvm
  labels:
    kubevirt.io/os: win2k12r2
```

The ``kubevirt.io/os`` label is based on the short OS identifier from [libosinfo](https://libosinfo.org/).
The following Short IDs are currently supported:

| Short ID      | Name                             | Version | Family | ID                                 |
|---------------|----------------------------------|---------|--------|------------------------------------|
| **fedora26**  | Fedora 26                        | 26      | linux  | http://fedoraproject.org/fedora/26 |
| **fedora27**  | Fedora 27                        | 27      | linux  | http://fedoraproject.org/fedora/27 |
| **rhel7.0**   | Red Hat Enterprise Linux 7.0     | 7.0     | linux  | http://redhat.com/rhel/7.0         |
| **rhel7.1**   | Red Hat Enterprise Linux 7.1     | 7.1     | linux  | http://redhat.com/rhel/7.1         |
| **rhel7.2**   | Red Hat Enterprise Linux 7.2     | 7.2     | linux  | http://redhat.com/rhel/7.2         |
| **rhel7.3**   | Red Hat Enterprise Linux 7.3     | 7.3     | linux  | http://redhat.com/rhel/7.3         |
| **rhel7.4**   | Red Hat Enterprise Linux 7.4     | 7.4     | linux  | http://redhat.com/rhel/7.4         |
| **win2k12r2** | Microsoft Windows Server 2012 R2 | 6.3     | winnt  | http://microsoft.com/win/2k12r2    |
| **win2k16**   | Microsoft Windows Server 2016    | 1709    | winnt  | http://microsoft.com/win/2k16      |

For the updated list please refer to kubeVirt user guide - [Guest Operating System Information](https://kubevirt.io/user-guide/virtual_machines/guest_operating_system_information/)


To get a full list of operating systems from Libosinfo database:

```
# List all operating systems
$ osinfo-query os

 Short ID             | Name                                              | Version | ID
----------------------------------------------------------------------------------------------------------------------
...
 fedora24             | Fedora 24                                      | 24        | http://fedoraproject.org/fedora/24
 fedora25             | Fedora 25                                      | 25        | http://fedoraproject.org/fedora/25
 fedora26             | Fedora 26                                      | 26        | http://fedoraproject.org/fedora/26
...
 win2k12r2          | Microsoft Windows Server 2012 R2 | 6.3       | http://microsoft.com/win/2k12r2
...
```

Libosinfo database can be queried to extract additional information about the operating system using the short-id.
Conditions allow filtering based on specific properties of an entity.
For example, to get specific properties for Microsoft Windows Server 2012, use


```
# Get OS specific info
$ osinfo-query os short-id=win2k12r2

 Short ID             | Name                                              | Version | ID
----------------------------------------------------------------------------------------------------------------------
 win2k12r2          | Microsoft Windows Server 2012 R2 | 6.3       | http://microsoft.com/win/2k12r2

```

Additional properties that can be queried include : name, version, family, vendor, release-date, eol-date, codename, id.
For more information please refer the osinfo-query man page.


The libosinfo database of valid short-ids/URIs is customizable by the host admin or user/application by simply dropping new XML files into a
defined location.
This is useful in case there is a need for custom short-ids/URIs for custom derived distros, or have obscure embedded OS distros for example.

While libosinfo provides a C library API, there is also support for applications just reading the XML files directly [(osinfo-db-tools)](https://gitlab.com/libosinfo/osinfo-db-tools/tree/master/docs) instead of a C library API binding to $PROGRAMMING-LANGUAGE.


### Use with presets

A Virtual Machine Preset representing an operating system with a ``kubevirt.io/os`` label could be applied on any given
Virtual Machine that have and match the``kubevirt.io/os`` label.
