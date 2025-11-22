#include "cxtypedefs.h"

// todo WIP
// v1, simple version one line ascii, one line hex
void print_binhex(void* adr, int len) {
	int byte_perline = 16;
	int width_perline = byte_perline*4 + 8;
	int linecnt = len/byte_perline + 1;
	int totlen =  len*50 + 1;
	char buf[totlen];
	memset(buf, ' ', totlen);
	// memset(buf, '+', totlen);
	buf[totlen-1] = 0;
	int pos = 0;
	uchar* ptr = (uchar*)adr;
	char bufascii[byte_perline+1];
	int posascii = 0;
	char bufhex[byte_perline*5+1];
	int poshex = 0;
	for (int idx=0; idx<len; idx++) {
		uchar tv = ptr[idx];
		int lineno = idx/byte_perline;
		int offhex = (idx%byte_perline)*3 + lineno * width_perline;
		int offascii = (lineno+1) * width_perline - byte_perline + (idx%byte_perline);
		// printf("at=%d lineno=%d offhex=%d offascii=%d val=%2x\n", idx, lineno, offhex, offascii, tv);
		// assert(offhex+4<totlen);
		if (idx%byte_perline==0 ) {
			poshex += snprintf(bufhex+poshex, sizeof(bufhex)-poshex-1, "%2d ", lineno);
			posascii += snprintf(bufascii+posascii, sizeof(bufascii)-posascii-1, "%2d ", lineno);
		}
		if (idx%byte_perline==0 && idx%8==0) {
			poshex += snprintf(bufhex+poshex, sizeof(bufhex)-poshex-1, "| ");
		}
		poshex += snprintf(bufhex+poshex, sizeof(bufhex)-poshex-1, "%02x ", tv);
		posascii += snprintf(bufascii+posascii, sizeof(bufascii)-posascii-1, "%c", isprint(tv)?tv:'.');
		if (idx%8==7) {
			poshex += snprintf(bufhex+poshex, sizeof(bufhex)-poshex-1, "| ");
		}
		if (idx>0 && idx%byte_perline==(byte_perline-1)) {
			pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%s\n", bufascii);
			pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%s\n", bufhex);
			posascii = poshex = 0;
		}
	}
	pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%s\n", bufascii);
	pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%s|\n", bufhex);
	posascii = poshex = 0;
	printf("^totol len %d, linecnt %d >>>\n%s$total len %d, linecnt %d <<<\n", len, linecnt, buf, len, linecnt);
}


// Source - https://stackoverflow.com/a
// Posted by Dúthomhas
// Retrieved 2025-11-20, License - CC BY-SA 3.0

// #include <stdio.h>
// #include <string.h>
// #include <stdlib.h>

static int pstrcmp_asc( const void* a, const void* b )
{
  return strcmp( *(const char**)a, *(const char**)b );
}
static int pstrcmp_desc( const void* a, const void* b )
{
  return -strcmp( *(const char**)a, *(const char**)b );
}

void cxstrs_sort(char** arr, int desc) {
    usize N = cstrs_len(arr);
    qsort(arr, N, sizeof(char*), desc ? pstrcmp_desc : pstrcmp_asc);
}

void cxstrs_reverse(char** arr) {

}

char* cxstrs_remove(char** arr, int idx) {
    return 0;
}


int cx_is_heap_ptr(void* ptr) {
	// GC_is_heap_ptr
	extern int GC_is_heap_ptr(void*);
	return GC_is_heap_ptr(ptr);
}

