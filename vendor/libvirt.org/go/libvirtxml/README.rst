=====================
libvirt-go-xml-module
=====================

.. image:: https://gitlab.com/libvirt/libvirt-go-xml-module/badges/master/pipeline.svg
   :target: https://gitlab.com/libvirt/libvirt-go-xml-module/pipelines
   :alt: Build Status
.. image:: https://img.shields.io/static/v1?label=godev&message=reference&color=00add8
   :target: https://pkg.go.dev/libvirt.org/go/libvirtxml
   :alt: API Documentation

Go API for manipulating libvirt XML documents

This package provides a Go API that defines a set of structs, annotated for use
with "encoding/xml", that can represent libvirt XML documents. There is no
dependency on the libvirt library itself, so this can be used regardless of
the way in which the application talks to libvirt.


Development status
==================

This API is considered to be production ready; note however that,
while unnecessary changes will be avoided, there are overall no
strong stability guarantees.

Please see the `VERSIONING <VERSIONING.rst>`_ file for information
about release schedule and versioning scheme.


Documentation
=============

* `API documentation for the bindings <https://pkg.go.dev/libvirt.org/go/libvirtxml>`_

* `Libvirt XML schema documentation <https://libvirt.org/format.html>`_

  * `capabilities <https://libvirt.org/formatcaps.html>`_
  * `domain <https://libvirt.org/formatdomain.html>`_
  * `domain capabilities <https://libvirt.org/formatdomaincaps.html>`_
  * `domain snapshot <https://libvirt.org/formatsnapshot.html>`_
  * `network <https://libvirt.org/formatnetwork.html>`_
  * `node device <https://libvirt.org/formatnode.html>`_
  * `nwfilter <https://libvirt.org/formatnwfilter.html>`_
  * `secret <https://libvirt.org/formatsecret.html>`_
  * `storage <https://libvirt.org/formatstorage.html>`_
  * `storage encryption <https://libvirt.org/formatstorageencryption.html>`_


Contributing
============

The libvirt project aims to add support for new XML elements to
libvirt-go-xml-module as soon as they are added to the main libvirt C
library. If you are submitting changes to the libvirt C library
that introduce new XML elements, please submit a libvirt-go-xml-module
change at the same time. Bug fixes and other improvements to the
libvirt-go-xml-module library are welcome at any time.

For more information, see the `CONTRIBUTING <CONTRIBUTING.rst>`_
file.
