# libvmi

libvmi is a VMI manifest composer.

## Overview

### Motivation

While reading code, especially e2e test code, a difficulty has been
observed to understand in an easy and clear manner, what is the content
of a VMI object. In a long chain of function call stack, different portions
of the VMI got updated, sometime overriding previously set fields.

### Goal
 - Simplify the creation of VMI objects.
 - Easily understand what VMI objects contain.

## How
libvmi is aimed to build the vmi manifest.

It uses the builder pattern, allowing a modular construction of VMI
different sections.

### Rules
In order to keep the package useful and easy to maintain, a few
rules are in order.

The main goal of these rules are to keep the package simple to use and
easy to grow. While exceptions may apply, they will need a wide consensus
and should not be overused. It would be better to just ask to change the rules.

- Use only the existing builder pattern to support editing fields.
- Place builders in a proper subject file, either one that exists or in a new
  one with a good name. A common developer should be able to find the relevant
  content based on the file name. 
  - Do **not** fill up the `vmi.go` file, unless you have a very good reason.
- Do **not** add logic in builders, unless they fell into these categories:
  - It is a factory that is widely used. (e.g. amount of memory depending on
    the architecture).
  - It is a widely used abstraction that combines several fields that have some
    dependency on each other.
- Any builder can be added if it has no logic (except for lazy creation of the
  path to the relevant fields).
  In practice, this implies that even builders that are used by a small amount
  of callers can be added, as long as they do not possess logic.
- Do **not** add commands on objects, e.g. calls through clients.
- Building general annotations or labels which have special meaning are a fit
  if they related to VM or VMI objects.
  To clarify, annotation that relate to pods, are not a perfect fit here.
  Annotations that relate to a specific subject (e.g. network) may fit here
  or under a more dedicated library/package.

> **Note**: A builder is considered `widely used` when it is needed from multiple
> packages. In case a single package or test file is using it, it may fit better
> under that package or test. The reason is simple, it has more context closer to
> the usage and known by the developers in a more accurate manner.

### Structure

- VMI: `vmi.go` contains the most basic tooling to start building VMI manifests.
  It contains the most basic factory (`New`), the definition of how the builders
  look like and any other helper that serves the whole libvmi package.
- Factory: `factory.go` file contains commonly used base VMI specs.
  Users will usually pick one factory to use and then add different
  builders to continue building the spec.
- Subject builders: Various files in which builders and defined.
  These files should group builders with some commonality, such that they
  can be easily found. With time, the grouping and naming may change
  (e.g. subjects split when they grow too much).

## Maintenance and Ownership

Maintainers are expected to follow the above rules or ask for exceptions.
Possibly asking to change a rule with a good reasoning.

Make sure to always keep things in focus, simple and clean.
When things get out of control, things get slower, not faster.
