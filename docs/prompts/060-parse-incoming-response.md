# Context
    - See /docs/specs/overview.md for the broader context
    - The existing code receives a response from the worldtides api
       as an IncomingResponse
    - A response shape back to the client is defined in overview.md

# Role and Objectives

- To map the response received from the worldtides into that required by overview.md, including validation

## Instructions
- Use a similar two-step processing approach to that used to map incoming requests. I.e. one semantic one in go model space, and another step to produce a fully formed HTTP response.
- Write suitable tests for the new code introduced here
- That may involve new tests in more than one scope
- Read and follow all my cursor skills that are concerned with testing

## Non instructions