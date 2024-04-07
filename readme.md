# agentic

agentic is a system to produce code according to specifications using LLMs

agent uses a local LLM-type autocompletion system to do several things.

(1) generate code / modify existing code.

(2) review changes, updating the code patches as needed.

(3) compile the code

(4) run the tests. if they do not pass, go to 2.

(5) verify that the code meets specification. If it does not, goto 2,
to done to _update_ the code, using the output of the verification.

# LLM setup

1. AI system

https://github.com/Ai00-X/ai00_server

2. Model

https://huggingface.co/BlinkDL/rwkv-6-world/blob/main/RWKV-x060-World-1B6-v2.1-20240328-ctx4096.pth

3. Convert as per instruction in AI00 docs.
