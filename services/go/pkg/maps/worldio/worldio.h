#ifndef WORLD_H
#define WORLD_H

#define _FILE_OFFSET_BITS 64

#include "tools.h"
#include "engine.h"
#include "texture.h"
#include "state.h"

void freeocta(cube *c);
cube *loadchildren_buf(void *p, size_t len, int size, int _mapversion);
size_t savec_buf(void *p, unsigned int len, cube *c, int size);

MapState *empty_world(int size);

MapState *partial_load_world(
        void *p,
        size_t len,
        int numvslots,
        int _worldsize,
        int _mapversion,
        int numlightmaps,
        int numpvs,
        int blendmap
);

size_t partial_save_world(
        void *p,
        size_t len,
        MapState *state,
        int _worldsize
);

bool load_texture_index(void *data, size_t len, MapState *state);


int getnumvslots(MapState *state);
VSlot *getvslotindex(MapState *state, int i);

cube *getcubeindex(cube *c, int i);
void cube_setedge(cube *c, int i, uchar value);
void cube_settexture(cube *c, int i, ushort value);
bool apply_messages(MapState *state, int _worldsize, void *data, size_t len);

#endif
