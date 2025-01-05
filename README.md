This was a hackday project aiming to create a pre-commit hook that would parse all go files through a locally run llm (via ollama) to check for best practices.

The initial testing revealed that models that were small enough to not require big download sizes and memory to run in a macbook were not accurate enough for this task.
