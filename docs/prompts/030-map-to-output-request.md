# Role

To provide some code that converts an incoming GET request into a new http request than can be relayed
to the worldapi server as described in /docs/specs/overview.md

## Objectives

- Write a DRY function with the following signature

```
func SynthesiseOutputRequest(*IncomingRequest) (*OutputRequest, error)
```

- The type OutputRequest is the logical model for the output request - not a native http request.


## Instructions

- Read the API documentation for https://www.worldtides.info/apidocs/extremes
- Design the logical process to map an IncomingRequest into a well formed output request to the worldtides api
- Code the OutputRequest go type
- Write a pure function to construct an OutputRequest from a given InputRequest
- Provide appropriate tests for the code generated in this step.

## Non instructions
- Do not generate any other code than that specified here.

