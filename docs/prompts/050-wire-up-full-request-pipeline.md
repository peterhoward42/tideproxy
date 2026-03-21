# Context

- The existing code:
    - defines required system behaviour in /docs/specs/overview.md
    - a placeholder implementation of Application.handleTides
    - a set of helpers that aim to make the implementation of handleTides a simple wiring up excercise.

# Role and Objectives

- Replace the placeholder implementation of handleTides with a real implementation
- It's scope is limited to the logic required to emit the real outbound request and to receive a response. But not to do anything yet with that response other than to check the response status code.

## Instructions
- The overview wiring up sequence should use the following in order:
    -  UnmarshalIncomingRequest
    -  SynthesiseOutputRequest
    -  BuildOutputHTTPRequest

- Invent any relevant external system interfaces that need to be available in the Dependencies structure DI into the application.
- Create test double fake implementation(s) of any new interfaces thus introduced
- Produce a suitable test suite for the new code created by this prompt
- Explicitly read and follow the guidelines in all my registered cursor skills concerned with generating test code.

## Non instructions

- Do not invent new code that has similar scope to the existing functions used above. If the existing functions
  are not fit for purpose as they are, upgrade them so that they can be used conveniently in the wiring up.