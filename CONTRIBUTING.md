# Contributing

## Pull Requests
There are a few things to consider before opening a pull request:

- Always open an issue first to discuss the intended change, then go for the PR, unless it's a trivial change.

- Keep the change set in your PR small. A PR should address a single concern, and not try to solve many issues at the same time.

- If your PR contains new functionality, it needs to add test cases for that.

- Open a PR only when you're done with the topic and your changes are ready for review. Don't keep adding commits to a PR you've just opened.

- Squash commits in your branch before opening the PR, unless there is a compelling reason to keep separate commits. The need for several commits in a PR is often a sign that the PR tries to address more than one concern. Split into several PRs in that case.

- Make your commit comments start with `#{issue number}: `, so they can be correlated with the according issues.

- Use `git commit --amend --date="$(date -R)"` when adding commits to your PR, and force push.

- If discussion is needed during implementation of a feature, refer to your branch in the associated issue and let's discuss there.

- If you still need to continue adding commits to a PR (sometimes that just happens), put the PR on hold by adding a comment to that effect, so that reviewers know they don't have to look at it yet.


## Coding Guidelines
Let's use common sense and be consistent. Quoting from [Google's C++ Style Guide](https://google.github.io/styleguide/cppguide.html#Parting_Words), since there's no better way of putting it:


> If you are editing code, take a few minutes to look at the code around you and determine its style. If they use spaces around their if clauses, you should, too. If their comments have little boxes of stars around them, make your comments have little boxes of stars around them too.

> The point of having style guidelines is to have a common vocabulary of coding so people can concentrate on what you are saying, rather than on how you are saying it. We present global style rules here so people know the vocabulary. But local style is also important. If code you add to a file looks drastically different from the existing code around it, the discontinuity throws readers out of their rhythm when they go to read it. Try to avoid this.
