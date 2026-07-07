package com.example.simple;

import com.example.simple.dto.SimpleRequest;

public class SimpleServiceImpl implements SimpleInterface {
    private final SimpleRequest request;

    public SimpleServiceImpl(SimpleRequest request) {
        this.request = request;
    }

    @Override
    public String process() {
        return doProcess();
    }

    private String doProcess() {
        return "done";
    }
}
