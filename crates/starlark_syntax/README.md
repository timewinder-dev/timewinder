Hard forked from https://github.com/facebookexperimental/starlark-rust to expose `AstModuleFields`


Updating this is going to be painful. 
I'd love to be a better open-source citizen and just add their hard work as a dependency. 
It's fabulous to not need to write the grammar, the parser, the lexer -- the whole bit -- and I'm willing to suffer the pain of updating against a changing API (or pinning, in worst-case).

However, the simplest path was just to bring it in and make importable the things they expressly hid. 
I respect that decision, even if I disagree with it, but it means a hard fork here to accelerate the project.
