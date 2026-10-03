package com.example.notesjava.image;

public interface ImageStore {

    String put(String key, byte[] data, String contentType);
}
