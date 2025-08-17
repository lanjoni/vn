# Coding Style Preferences

We're following default Go preferences and coding style guide.

## Comments
- Do not add any comments to the code, just in case to simplify understandment 
  - like when comments about size - 512kb etc

## Code Style
- Follow existing patterns in the codebase
- Maintain consistency with existing scanner implementations
- Prefer tabs over spaces

## Tests
- Unit tests are on the same level as the original file to test
- Integration and E2E tests are available at `/tests`
- There's a vulnerable server provided at `/test-server`
  - It only has mock data to emulates a vulnerable server to test
- You can run `make test` to run all tests
- You can see more at `docs/tests.md` and `docs/test-server.md`
