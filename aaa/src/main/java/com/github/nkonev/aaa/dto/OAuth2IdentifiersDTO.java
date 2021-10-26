package com.github.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonAutoDetect;
import com.fasterxml.jackson.annotation.JsonTypeInfo;

import java.io.Serializable;

@JsonTypeInfo(use = JsonTypeInfo.Id.CLASS, include = JsonTypeInfo.As.PROPERTY, property = "@class")
@JsonAutoDetect(fieldVisibility = JsonAutoDetect.Visibility.ANY, getterVisibility = JsonAutoDetect.Visibility.NONE, setterVisibility = JsonAutoDetect.Visibility.NONE, isGetterVisibility = JsonAutoDetect.Visibility.NONE)
public record OAuth2IdentifiersDTO  (
    String facebookId,
    String vkontakteId,
    String googleId
) implements Serializable {
    public OAuth2IdentifiersDTO() {
        this(null, null, null);
    }

    public OAuth2IdentifiersDTO withGoogleId(String newGoogleId) {
        return new OAuth2IdentifiersDTO(
                facebookId,
                vkontakteId,
                newGoogleId
        );
    }

    public OAuth2IdentifiersDTO withVkontakteId(String newVkontakteId) {
        return new OAuth2IdentifiersDTO(
                facebookId,
                newVkontakteId,
                googleId
        );
    }

    public OAuth2IdentifiersDTO withFacebookId(String newFacebookId) {
        return new OAuth2IdentifiersDTO(
                newFacebookId,
                vkontakteId,
                googleId
        );
    }
}
