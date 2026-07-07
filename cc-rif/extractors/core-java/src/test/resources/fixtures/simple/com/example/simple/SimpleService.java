package com.example.simple;

import com.example.simple.dto.SimpleRequest;

public class SimpleService {
    private final SimpleRequest request;

    public SimpleService(SimpleRequest request) {
        this.request = request;
    }

    public String process() {
        return validate();
    }

    private String validate() {
        return "ok";
    }
}
