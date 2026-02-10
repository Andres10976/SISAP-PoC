vi.stubGlobal(
  "fetch",
  vi.fn(() => {
    throw new Error("fetch must be mocked in each test");
  }),
);
