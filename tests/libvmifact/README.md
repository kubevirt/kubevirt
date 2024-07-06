# libvmifact

libvmifact is a VMI manifest factory.

## Overview

### Motivation

While reading code, especially test code, a difficulty has been
observed to understand in an easy and clear manner, what is the content
of a VMI object. In a long chain of function call stack, different portions
of the VMI got updated, sometime overriding previously set fields.

### Goal
 - Simplify the creation of VMI objects.
 - Easily understand what VMI objects contain.

## How
libvmifact is aimed to build the vmi manifest with a predefined set of settings
that are commonly used by the e2e tests.

It uses the `libvmi` package, allowing a modular construction of VMI
different sections.

### Rules
In order to keep the package useful and easy to maintain, a few
rules are in order.

The main goal of these rules are to keep the package simple to use and
easy to grow. While exceptions may apply, they will need a wide consensus
and should not be overused. It would be better to just ask to change the rules.

- Do **not** add logic in factories, unless they fell into these categories:
  - It is a factory that is widely used. (e.g. amount of memory depending on
    the architecture).
  - It is a widely used abstraction that combines several fields that have some
    dependency on each other.
- Do **not** add commands on objects, e.g. calls through clients.

> **Note**: A factory is considered `widely used` when it is needed from multiple
> packages. In case a single package or test file is using it, it may fit better
> under that package or test. The reason is simple, it has more context closer to
> the usage and known by the developers in a more accurate manner.

### Structure

- Factory: `factory.go` file contains commonly used base VMI specs.
  Users will usually pick one factory to use and then add different
  builders to continue building the spec.

## Maintenance and Ownership

Maintainers are expected to follow the above rules or ask for exceptions.
Possibly asking to change a rule with a good reasoning.

Make sure to always keep things in focus, simple and clean.
When things get out of control, things get slower, not faster.
