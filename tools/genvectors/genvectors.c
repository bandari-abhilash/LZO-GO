/* genvectors - regenerate the LZO1Z test vectors in testdata/ using liblzo2.
 *
 * Copyright (C) 2026 Abhilash
 *
 * This program is free software; you can redistribute it and/or modify it
 * under the terms of the GNU General Public License as published by the Free
 * Software Foundation; either version 2 of the License, or (at your option)
 * any later version.  See the LICENSE file at the repository root.
 *
 * Build:  cc -O2 -o genvectors genvectors.c -llzo2
 * Run:    ./genvectors ../../testdata
 *
 * Each corpus is written as <name>.orig (the plaintext) and <name>.lzo1z
 * (the lzo1z_999_compress output).  All corpora are generated from a fixed
 * LCG seed, so the .orig files are byte-identical on every machine; the
 * .lzo1z files depend on your liblzo2 version.
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <lzo/lzo1z.h>
#include <lzo/lzoconf.h>

static unsigned long lcg_state = 12345UL;
static unsigned char lcg_byte(void)
{
    lcg_state = lcg_state * 1103515245UL + 12345UL;
    return (unsigned char)((lcg_state >> 16) & 0xFF);
}

static const char *TEXT =
    "the quick brown fox jumps over the lazy dog. ";

/* Fill buf with one of the named corpora; returns the length used. */
static size_t build(const char *name, unsigned char *buf, size_t cap)
{
    size_t i, n;
    if (!strcmp(name, "tiny1"))  { buf[0] = 'A'; return 1; }
    if (!strcmp(name, "tiny4"))  { memcpy(buf, "AAAA", 4); return 4; }
    if (!strcmp(name, "tiny18")) { memcpy(buf, "ABCDEFGHIJKLMNOPQR", 18); return 18; }

    if (!strcmp(name, "zeros"))  { n = 16384; memset(buf, 0, n); return n; }

    if (!strcmp(name, "random")) {           /* incompressible */
        n = 4096;
        for (i = 0; i < n; i++) buf[i] = lcg_byte();
        return n;
    }
    if (!strcmp(name, "text")) {             /* long repeating matches */
        size_t tl = strlen(TEXT);
        n = 7991;
        for (i = 0; i < n; i++) buf[i] = (unsigned char)TEXT[i % tl];
        return n;
    }
    if (!strcmp(name, "records")) {          /* fixed-width records, near-identical */
        const size_t rec = 116, count = 200;
        n = rec * count;
        for (i = 0; i < count; i++) {
            unsigned char *r = buf + i * rec;
            memset(r, ' ', rec);
            memcpy(r, "SYMBOL00000", 11);
            r[9]  = (unsigned char)('0' + (i / 10) % 10);
            r[10] = (unsigned char)('0' + i % 10);
            memcpy(r + 16, "0000107.15", 10);   /* price   */
            memcpy(r + 32, "000000000250", 12); /* qty     */
            r[rec - 1] = '\n';
        }
        return n;
    }
    if (!strcmp(name, "mixed")) {            /* alternating compressible/random runs */
        size_t tl = strlen(TEXT);
        n = 12000;
        for (i = 0; i < n; i++) {
            size_t phase = (i / 500) % 3;
            if (phase == 0)      buf[i] = 0;
            else if (phase == 1) buf[i] = (unsigned char)TEXT[i % tl];
            else                 buf[i] = lcg_byte();
        }
        return n;
    }
    if (!strcmp(name, "large")) {            /* > 64 KiB of history, long distances */
        n = 60000;
        for (i = 0; i < n; i++)
            buf[i] = (i % 1021 == 0) ? lcg_byte() : (unsigned char)(i % 251);
        return n;
    }
    (void)cap;
    fprintf(stderr, "unknown corpus %s\n", name);
    exit(1);
}

int main(int argc, char **argv)
{
    static const char *names[] = {
        "tiny1", "tiny4", "tiny18", "text", "zeros",
        "random", "records", "mixed", "large", NULL
    };
    const char *dir = (argc > 1) ? argv[1] : "testdata";
    unsigned char *in, *out, *wrkmem;
    size_t cap = 1 << 20;
    int i;

    if (lzo_init() != LZO_E_OK) { fprintf(stderr, "lzo_init failed\n"); return 1; }

    in     = malloc(cap);
    out    = malloc(cap + cap / 16 + 64 + 3);
    wrkmem = malloc(LZO1Z_999_MEM_COMPRESS);
    if (!in || !out || !wrkmem) { fprintf(stderr, "oom\n"); return 1; }

    for (i = 0; names[i]; i++) {
        char path[512];
        lzo_uint in_len = (lzo_uint)build(names[i], in, cap), out_len = 0;
        FILE *f;
        int r = lzo1z_999_compress(in, in_len, out, &out_len, wrkmem);
        if (r != LZO_E_OK) { fprintf(stderr, "%s: compress failed (%d)\n", names[i], r); return 1; }

        snprintf(path, sizeof path, "%s/%s.orig", dir, names[i]);
        if (!(f = fopen(path, "wb"))) { perror(path); return 1; }
        fwrite(in, 1, in_len, f); fclose(f);

        snprintf(path, sizeof path, "%s/%s.lzo1z", dir, names[i]);
        if (!(f = fopen(path, "wb"))) { perror(path); return 1; }
        fwrite(out, 1, out_len, f); fclose(f);

        printf("%-8s %6lu -> %6lu bytes\n", names[i],
               (unsigned long)in_len, (unsigned long)out_len);
    }
    printf("liblzo2 %s\n", lzo_version_string());
    return 0;
}
