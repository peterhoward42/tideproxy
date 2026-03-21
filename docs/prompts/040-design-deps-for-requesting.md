# Role

- The existing code includes the OutputRequest type to model semantically an API request
to the worldtides API. See /docs/specs/overview.md for the wider context.
- This prompt is to work out (but not to generate yet) how to build a real http request to wrap the OutputRequest in.
- The new builder will be manifested as a function that generates the appropriate HTTP request from
   a given OutputRequest.
- Our working out aims to help us identify any Dependency interfaces required in the existing Dependencies struct that will
  allow tests to be written for the new function using appropriately designed fake test double implementations of the new dependency interfaces
- It may be that no new dependency interfaces are required - that is a valid conclusion.

## Objectives


## Instructions
- Perform the analysis described above, and recommend what external system interfaces are required - if any.


## Non instructions
- Do not generate any code yet.

