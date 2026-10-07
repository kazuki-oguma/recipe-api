"use strict";
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

const expect = require("chai").expect;
const yaml = require('js-yaml');
const fs = require('fs');
const validUrl = require("valid-url");

let chai = require('chai');
const chaiHttp = require('chai-http');
chai.use(chaiHttp);

let env;
try {
    env = yaml.safeLoad(fs.readFileSync('env.yml', 'utf8'));
} catch (e) {
    console.log(e);
}

let BASE_URL = env.BASE_URL;


describe("", () => {
  before("初期化中...", async function() {
    this.timeout(20000);
    let wakeUp = true;
    let wakeUpLimit = setTimeout(function() {
      wakeUp = false;
    }, 15000);
    while(wakeUp) {
      await chai.request(BASE_URL)
        .get("/")
        .catch(function(res) {
            if (typeof res.response !== "undefined") {
            console.log("OK");
            wakeUp = false;
          }
        });
    }
  });

  describe("[基本実装] codecheck.yml", () => {
    it("BASE_URLに有効なURLが代入されている", () => {
      console.log(BASE_URL);
      // "記述されたURLが無効な形式です。"
      expect(validUrl.isUri(BASE_URL)).to.be.ok;
    });
  });

  describe("[基本実装] API server", () => {
    it("BASE_URL にアクセスするとコード404が返る", function(done) {
       this.timeout(6000);
      chai.request(BASE_URL)
      .get("/")
      .catch(function (res) {
        expect(res.status).to.equal(404);
        done();
      });
    });
  });
});