FROM emscripten/emsdk:1.40.0
WORKDIR /cube2

build:
    COPY cube2 cube2
    RUN cd cube2/src/web && emmake make client -j8
    SAVE ARTIFACT cube2/bb.html  AS LOCAL build/bb.html
    SAVE ARTIFACT cube2/game  AS LOCAL build/game
    SAVE ARTIFACT cube2/js  AS LOCAL build/js
    SAVE ARTIFACT cube2/*.js  AS LOCAL build/
    SAVE ARTIFACT cube2/*.wasm  AS LOCAL build/
    SAVE ARTIFACT cube2/*.data  AS LOCAL build/
