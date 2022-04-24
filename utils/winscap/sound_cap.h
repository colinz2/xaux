#ifndef _SOUND_CAP_H_
#define _SOUND_CAP_H_

struct SCConfig {
    unsigned short channels;
    unsigned short sampleRate;
    unsigned short bits;
};

typedef void *SCHandle;

typedef void (*VOICE_BUFFER_CB)(unsigned char* data, int len, void* prv);


// 初始化
int sound_cap_init(struct SCConfig *scConfig, SCHandle* handle);

// 采样
int sound_cap_run_loop(SCHandle handle, VOICE_BUFFER_CB cb, void* prv);



#endif
