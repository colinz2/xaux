#ifndef _WINSCAP_H_
#define _WINSCAP_H_

#include "sound_cap.h"

typedef void *WINSCAP_HANDLE;

int winscap_init(struct SCConfig *config, WINSCAP_HANDLE* handle);

int winscap_run(WINSCAP_HANDLE handle, VOICE_BUFFER_CB cb, void *prv);



#endif