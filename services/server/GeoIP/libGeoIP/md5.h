#include "types.h"

typedef struct {
  u32 A,B,C,D;          /* chaining variables */
  u32  nblocks;
  byte buf[64];
  int  count;
} MD5_CONTEXT;

void md5_init( MD5_CONTEXT *ctx );
void md5_write( MD5_CONTEXT *hd, byte *inbuf, size_t inlen);
void md5_final( MD5_CONTEXT *hd );

