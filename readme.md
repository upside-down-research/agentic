# agentic

agentic is a system to produce code according to specifications using LLMs. It is an "AI Software Engineer"

agent uses (in theory) a local LLM-type autocompletion system to do several things.

(1) generate code / modify existing code.

(2) review changes, updating the code patches as needed.

(3) compile the code

(4) run the tests. if they do not pass, go to 2.

(5) verify that the code meets specification. If it does not, goto 2,
to done to _update_ the code, using the output of the verification.

# what do we have now

We have a Go tool that will interact with AI00, Anthropic Claude, and OpenAI GPT.

1. It will plan a project and review the plan for correctness.
2. It will generate code and review the code for correctness.
3. It will **not** compile the code or run the tests.
4. It does **not** handle diffs.

The project was tied off at v0.1.0 because the generated code quality was simply too low. The next step would have to be
integrating a per-module review and patch process (as opposed to the usual 'here, have a all new function' approach). 


Entry point of use!

Set OPENAI_KEY to your key. I recommend OpenAI because it is the best and is the cheapest between it and Claude.

`make  && ./output/agentic --llm=openai --model=gpt-4-turbo examples/andon.in --output planning`

(Plausibly, RWKV or another local LLM can be done cheaper, but I don't have the machine for the big LLM model runnings).

# RWKV setup

1. AI system https://github.com/Ai00-X/ai00_server
2. Model https://huggingface.co/BlinkDL/rwkv-6-world/blob/main/RWKV-x060-World-1B6-v2.1-20240328-ctx4096.pth
3. Convert as per instruction in AI00 docs.
4. Install the model as per AI00 docs
5. cargo run --release

# contributions etc

The code and prompts are AGPL3 - they are free to use, but if you use them, you must share your code, even if 
behind a service.

But you're welcome to contribute!