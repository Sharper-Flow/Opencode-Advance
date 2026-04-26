// @bun
// src/index.ts
var src_default = {
  sessionCreated: async (session, _output) => {
    console.log("[oca] sessionCreated:", session.id);
  },
  sessionDeleted: async (session, _output) => {
    console.log("[oca] sessionDeleted:", session.id);
  }
};
export {
  src_default as default
};
