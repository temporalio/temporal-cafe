# Temporal Cafe

Temporal Cafe is an example application written to demonstrate how a Temporal application might be structured.

The main branch of this repo contains a basic version of an application which could be used as a backend system for an imaginary cafe. The cafe analogy has been chosen in order to reduce the need for much in the way of domain expertise in any specific field for readers.

The basic version includes examples of core Temporal primitives.

More complex features, design patterns (or even alternatives to Temporal as the backend, for comparison) are provided via branches and visible via Pull Requests.

This README will list the various feature/comparison PRs as they are considered ready for readers to digest.

Please feel free to suggest features or patterns you would like us to cover by filing issues.

## Features

* [Workflow Update](https://github.com/temporalio/temporal-cafe/pull/1): Demonstrates the use of the Workflow Update feature instead of using Signals. Of particular note here is the ability to validate the contents of the update and provide a synchronous response from the client's point of view.