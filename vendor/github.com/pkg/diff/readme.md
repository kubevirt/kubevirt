Development on this package is on hold until there is clarity on whether generics will be added for Go 2. Please check back in later.

See https://godoc.org/github.com/pkg/diff for usage and docs.

License: BSD 3-Clause.

The remainder of this doc is for contributors.

Useful background reading about diffs:

* [Neil Fraser's website](https://neil.fraser.name/writing/diff)
* [Myers diff paper](http://www.xmailserver.org/diff2.pdf)
* [Guido Van Rossum's reverse engineering of the unified diff format](https://www.artima.com/weblogs/viewpost.jsp?thread=164293)

TODO before declaring this package stable:

* API review. Some open questions:
  - Pair won't suffice for other diff algorithms, like patience diff or using indentation-based heuristics. Writer/WriteOpts might not suffice for other writing formats. We'll probably just need to cross that bridge when we get to it, but our current names might be too general.
  - Should there be some way to step through an EditScript manually? E.g. for use in a system that uses the diff to perform actions to get to a desired end state.
  - I've long wanted a way to vary the number of lines of context depending on content. For example, if I deleted the first line of a function, I don't need three full lines of "before" context; it should truncate at the function declaration. Do we have enough state to store such information, if needed? (Would it be better to make EditScript an interface?)
  - What is the compatibility guarantee? Do we guarantee the exact same diff given the same input? If not, it is less useful for other peoples' golden tests, but it really ties the hands of the package authors; it'd be nice to be able to generate better diffs over time.
  - Add some test helpers to make it easy for people to write nice test failure output? Does this place new demands on EditScripts, like the ability to detect complete replacement?
* Get some miles on the code.
* Run through the TODOs scattered throughout the code and decide which if any need action soon.
* Do some fuzzing?
* Do we need copyright headers at top of files?
