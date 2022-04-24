#include "winscap.h"
#include <stdio.h>


void dataCallBack(unsigned char* data, int len, void* prv) {
    
    printf("len = %d \n", len);

}

int main() {
    SCConfig config;
    config.bits = 16;
    config.channels = 2;
    config.sampleRate = 48000;

    
    WINSCAP_HANDLE handle = nullptr;
    winscap_init(&config, &handle);
    winscap_run(handle, dataCallBack, nullptr);
    return 0;
}

