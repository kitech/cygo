#include <ffi.h>

int ffi_get_default_abi() { return FFI_DEFAULT_ABI; }
int ffi_type_size() { return sizeof(ffi_type); }
int ffi_cif_size() { return sizeof(ffi_cif); }

void dump_pointer_array(int n, void** ptr) {
    for (int i = 0;i < n; i ++) {
        printf("%p %d, = %p\n", ptr, i, ptr[i]);
    }
}

void**pointer_array_new(int n) { return (void**)malloc(sizeof(void*)*n); }
void pointer_array_set(void**ptr, int idx, void*val) { ptr[idx] = val;}
void* pointer_array_get(void**ptr, int idx) { return ptr[idx]; }
void pointer_array_free(void*ptr) {free(ptr);}
void** pointer_array_addr(void**ptr) { return &ptr[0]; }
