const defaultResolution = 'h720';

export const KEY_VIDEO_RESOLUTION = 'videoResolution2';
export const KEY_SCREEN_RESOLUTION = 'screenResolution2';

export const getVideoResolution = () => {
    let got = localStorage.getItem(KEY_VIDEO_RESOLUTION);
    if (!got) {
        localStorage.setItem(KEY_VIDEO_RESOLUTION, defaultResolution);
        got = localStorage.getItem(KEY_VIDEO_RESOLUTION);
    }
    return got;
}

export const getScreenResolution = () => {
    let got = localStorage.getItem(KEY_SCREEN_RESOLUTION);
    if (!got) {
        localStorage.setItem(KEY_SCREEN_RESOLUTION, defaultResolution);
        got = localStorage.getItem(KEY_SCREEN_RESOLUTION);
    }
    return got;
}

export const setVideoResolution = (newVideoResolution) => {
    localStorage.setItem(KEY_VIDEO_RESOLUTION, newVideoResolution);
}

export const setScreenResolution = (newVideoResolution) => {
    localStorage.setItem(KEY_SCREEN_RESOLUTION, newVideoResolution);
}


export const KEY_VIDEO_PRESENTS = 'videoPresents';
export const KEY_AUDIO_PRESENTS = 'audioPresents';

export const getStoredVideoDevicePresents = () => {
    let v = JSON.parse(localStorage.getItem(KEY_VIDEO_PRESENTS));
    if (v === null) {
        console.log("Resetting video presents to default");
        setStoredVideoPresents(true);
        v = JSON.parse(localStorage.getItem(KEY_VIDEO_PRESENTS));
    }
    return v;
}

export const setStoredVideoPresents = (v) => {
    localStorage.setItem(KEY_VIDEO_PRESENTS, JSON.stringify(v));
}

export const getStoredAudioDevicePresents = () => {
    let v = JSON.parse(localStorage.getItem(KEY_AUDIO_PRESENTS));
    if (v === null) {
        console.log("Resetting audio presents to default");
        setStoredAudioPresents(true);
        v = JSON.parse(localStorage.getItem(KEY_AUDIO_PRESENTS));
    }
    return v;
}

export const setStoredAudioPresents = (v) => {
    localStorage.setItem(KEY_AUDIO_PRESENTS, JSON.stringify(v));
}

export const KEY_LANGUAGE= 'language';

export const getStoredLanguage = () => {
    let v = JSON.parse(localStorage.getItem(KEY_LANGUAGE));
    if (v === null) {
        console.log("Resetting language to default");
        setStoredLanguage('en');
        v = JSON.parse(localStorage.getItem(KEY_LANGUAGE));
    }
    return v;
}

export const KEY_VIDEO_SIMULCAST = 'videoSimulcast';
export const KEY_SCREEN_SIMULCAST = 'screenSimulcast';

export const getStoredVideoSimulcast = () => {
    let v = JSON.parse(localStorage.getItem(KEY_VIDEO_SIMULCAST));
    if (v === null) {
        console.log("Resetting video simulcast to default");
        setStoredVideoSimulcast(true);
        v = JSON.parse(localStorage.getItem(KEY_VIDEO_SIMULCAST));
    }
    return v;
}

export const setStoredVideoSimulcast = (v) => {
    localStorage.setItem(KEY_VIDEO_SIMULCAST, JSON.stringify(v));
}

export const getStoredScreenSimulcast = () => {
    let v = JSON.parse(localStorage.getItem(KEY_SCREEN_SIMULCAST));
    if (v === null) {
        console.log("Resetting screen simulcast presents to default");
        setStoredScreenSimulcast(true);
        v = JSON.parse(localStorage.getItem(KEY_SCREEN_SIMULCAST));
    }
    return v;
}

export const setStoredScreenSimulcast = (v) => {
    localStorage.setItem(KEY_SCREEN_SIMULCAST, JSON.stringify(v));
}

export const KEY_ROOM_DYNACAST = 'roomDynacast';

export const getStoredRoomDynacast = () => {
    let v = JSON.parse(localStorage.getItem(KEY_ROOM_DYNACAST));
    if (v === null) {
        console.log("Resetting video dynacast to default");
        setStoredRoomDynacast(true);
        v = JSON.parse(localStorage.getItem(KEY_ROOM_DYNACAST));
    }
    return v;
}

export const setStoredRoomDynacast = (v) => {
    localStorage.setItem(KEY_ROOM_DYNACAST, JSON.stringify(v));
}

export const KEY_ROOM_ADAPTIVE_STREAM = 'roomAdaptiveStream';

export const getStoredRoomAdaptiveStream = () => {
    let v = JSON.parse(localStorage.getItem(KEY_ROOM_ADAPTIVE_STREAM));
    if (v === null) {
        console.log("Resetting adaptive stream to default");
        setStoredRoomAdaptiveStream(true);
        v = JSON.parse(localStorage.getItem(KEY_ROOM_ADAPTIVE_STREAM));
    }
    return v;
}

export const setStoredRoomAdaptiveStream = (v) => {
    localStorage.setItem(KEY_ROOM_ADAPTIVE_STREAM, JSON.stringify(v));
}


export const setStoredLanguage = (v) => {
    localStorage.setItem(KEY_LANGUAGE, JSON.stringify(v));
}

export const KEY_CHAT_EDIT_MESSAGE_DTO = 'chatEditMessageDto';

export const getStoredChatEditMessageDto = (chatId, defVal) => {
    let v = JSON.parse(localStorage.getItem(KEY_CHAT_EDIT_MESSAGE_DTO + '_' + chatId));
    if (v === null) {
        return defVal;
    }
    return v;
}

export const setStoredChatEditMessageDto = (v, chatId) => {
    localStorage.setItem(KEY_CHAT_EDIT_MESSAGE_DTO + '_' + chatId, JSON.stringify(v));
}

export const removeStoredChatEditMessageDto = (chatId) => {
    localStorage.removeItem(KEY_CHAT_EDIT_MESSAGE_DTO + '_' + chatId);
}
