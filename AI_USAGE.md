## AI_USAGE 

I used AI tools during this addignment mainly for learning ,debugging and for implementation.

## Tools Used

### ChatGPT
I used ChatGPT to:
- Understand the assignment requirements.
- Learn Go concepts while implementing the backend.
- Understand API design, request/response validatation ,concurrent ,and synchronization.
- Discuss debugging approaches and verify my understanding of the implementation.
- Review parts of my work while solving the assignment.

### Claude

I used Claude to:
- Get explanations for some Go/backend concepts.
- Discuss implementation approaches and debugging ideas.
- Review parts of my work while solving the assignent.

### Opencode 
I used Opencode while working on the project:
- Implement the starter code and understand the existing features.
- Help implement and debug parts of the backend.
- Investigate the four bugs part of the backend.
- Help verify the program output and behaviour.


## AI changes I changed or rejected 

Durig the debugging task, an AI- assisted change accidentally changed  the `opened` Counter logic while fixing the concurrency issue.checked the change against the original code and expected output ,noticed that it was incorrect,and changed it back so that each event type updated the correct answer.

I also close to keep the implemetation simple instead od adding unnecessary architecture or extra features suggested during discussion,because the assignment specifically values a solution that I can understand and explain.

## One thing AI helped me understand 

AI helped me understand the concurrency problem in the debugging program.Multiple workers were updating the same campaign statistics at the same time ,which could create a data race.I understood why synchronization was needed and verified the fix using Go's race detector.


## AI -generated sork that i had to debug 

Some AI-assisted code changes need to be checked and corrected before I accepted them. In particular ,while fu\ixing the debugging program,an incorrect change caused the `opened` counter to be updated in the wrong event case.I indentified the problem b comparing the code and output with the expected behaviour and corrected it .

I accepted AI suggestions as assistance rather than blindly accepting the generated code. I tested the implementation and made sure understood the final code before keeping it.
