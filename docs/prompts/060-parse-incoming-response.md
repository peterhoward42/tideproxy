- The existing code:
    - defines required system behaviour in /docs/specs/overview.md
    - receives an incoming response in Application.handleTides - but doesn't do anything with it yet

# Role and Objectives

- Model the expected response by creating an IncomingResponse struct
- Create a DRY function to parse and validate the IncomingResponse as being fit for purpose to serve the needs defined in the overview.md
- Add this processing step to handleTides

## Instructions
- Code the IncomingResponse struct
- Code the DRY parse and validate function
- Write suitable tests for the new code introduced here
- That may involve new tests in more than one scope
- Read and follow all my cursor skills that are concerned with testing

## Non instructions