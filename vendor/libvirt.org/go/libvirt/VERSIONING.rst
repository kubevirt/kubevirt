============================================
Versioning information for libvirt-go-module
============================================

Release schedule
================

The Go bindings follow the same `release schedule`_ as the main C
library, with new releases of both usually being tagged at the same
time.

.. _release schedule: https://libvirt.org/downloads.html#schedule


Versioning scheme
=================

Despite the release schedule being the same, the Go bindings do
**not** follow the same `versioning scheme`_ as the main C library.

The Go bindings have adopted `semantic versioning`_, which is both
expected in the Go ecosystem and extremly important in order to work
correctly within the Go module system.

When it's time to tag a new release, the logic described below is
followed: in this example, we will assume that the most recent
release of the Go bindings is ``v1.7005.0`` (made along libvirt
7.5.0) and that libvirt 7.6.0 has just been tagged.

* if libvirt 7.6.0 introduces new APIs

  * make sure the Go bindings implement them and tag the result as
    ``v1.7006.0``

* if libvirt 7.6.0 doesn't introduce new APIs

  * if there have been other tweaks and changes to the Go bindings
    since ``v1.7005.0``

    * tag the current code as ``v1.7005.1``

  * if the Go bindings are completely unchanged from ``v1.7005.0``

    * do nothing

This versioning scheme has the following desirable properties:

* it complies with the semantic versioning specification;

* it contains an encoded version of the libvirt API it implements,
  making it easy to tell at a glance whether or not the libvirt
  functionality you're interested in will be available to your Go
  application;

* it removes the need for users to update their import paths once per
  year even though the Go bindings have retained complete backwards
  compatibility;

* it avoids the situation where a new version of the Go bindings is
  tagged even though it contains no actual changes, as well as the
  opposite scenario where fixes made to the Go bindings cannot make
  it into a release until the C library introduces a new API.

.. _versioning scheme: https://libvirt.org/downloads.html#numbering
.. _semantic versioning: https://semver.org/
