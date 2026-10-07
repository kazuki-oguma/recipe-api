"use strict";
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

const expect = require("chai").expect;
const yaml = require('js-yaml');
const fs = require('fs');


let chai = require('chai');
const chaiHttp = require('chai-http');
chai.use(chaiHttp);
let should = chai.should();

let env;
try {
    env = yaml.safeLoad(fs.readFileSync('env.yml', 'utf8'));
} catch (e) {
    console.log(e);
}

let BASE_URL = env.BASE_URL;
let POST_RECIPE_PATH    = "/recipes";
let GET_ALL_RECIPE_PATH = "/recipes";
let GET_ONE_RECIPE_PATH = "/recipes/";
let UPDATE_RECIPE_PATH  = "/recipes/";
let DELETE_RECIPE_PATH  = "/recipes/";


let recipeList = [];

describe("", () => {
  describe('[基本実装] /POST recipes', () => {
    it('リクエストの必須パラメータが欠けている場合にレシピを作成できない', (done) => {
      let recipe = {
      }
      chai.request(BASE_URL)
      .post(POST_RECIPE_PATH)
      .send(recipe)
      .end((err, res) => {
        res.should.have.status(200);
        res.body.should.be.a('object');
        res.body.should.have.property('message').eql('Recipe creation failed!');
        done();
      });
    });
    it('レシピを作成できる', (done) => {
      let recipe = {
        "title": "トマトスープ",
        "making_time": "15分",
        "serves": "5人",
        "ingredients": "玉ねぎ, トマト, スパイス, 水",
        "cost": 450
      }
      chai.request(BASE_URL)
      .post(POST_RECIPE_PATH)
      .send(recipe)
      .end((err, res) => {
        res.should.have.status(200);
        res.body.should.be.a('object');
        res.body.should.have.property('message').eql('Recipe successfully created!');
        res.body.should.have.property('recipe').be.a('Array');
        res.body.recipe.length.should.eql(1);
        done();
      });
    });
  });

  describe('[基本実装] /GET recipes', () => {
    it('全てのレシピを取得できる', (done) => {
      chai.request(BASE_URL)
      .get(GET_ALL_RECIPE_PATH)
      .end((err, res) => {
        expect(res).to.have.status(200);
        expect(res).to.be.a('object');
        for (let i=0; i < res.body.recipes.length; i++) {
          expect(res.body.recipes[i]).to.have.property('id');
          recipeList.push(res.body.recipes[i]);
        }
        done();
      });
    });
  });

  describe('[基本実装] /GET/{id} recipe', () => {
    it('idで指定したレシピを取得できる', (done) => {
      const targetId = (recipeList.length > 0 ? recipeList[0].id : 1);
      chai.request(BASE_URL)
      .get(GET_ONE_RECIPE_PATH + targetId)
      .end((err, res) => {
        res.should.have.status(200);
        res.body.should.be.a('object');
        res.body.should.have.property('message').eql('Recipe details by id');
        res.body.should.have.property('recipe').be.a('Array');
        res.body.recipe.length.should.eql(1);
        const targetRecipe = (recipeList.length > 0 ? recipeList[0] : res.body.recipe[0]);
        res.body.recipe[0].should.have.property('id').eql(targetRecipe.id);
        res.body.recipe[0].should.have.property('title').eql(targetRecipe.title);
        res.body.recipe[0].should.have.property('serves').eql(targetRecipe.serves);
        res.body.recipe[0].should.have.property('ingredients').eql(targetRecipe.ingredients);
        res.body.recipe[0].should.have.property('cost').eql(targetRecipe.cost);
        done();
      });
    });
  });

  describe('[基本実装] /PATCH/{id} recipe', () => {
    it('レシピを更新できる', (done) => {
      const recipe = {
        "title": "オムライス",
        "making_time": "20分",
        "serves": "7人",
        "ingredients": "玉ねぎ,卵,スパイス,醤油",
        "cost": 400
      }
      const targetId = (recipeList.length > 0 ? recipeList[0].id : 1);
      chai.request(BASE_URL)
      .patch(UPDATE_RECIPE_PATH + targetId)
      .send(recipe)
      .end((err, res) => {
        res.should.have.status(200);
        res.body.should.be.a('object');
        res.body.should.have.property('message');
        done();
      });
    });
  });

  describe('[基本実装] /DELETE/{id} recipe', () => {
    it('レシピを削除できる', (done) => {
      const targetId = (recipeList.length > 0 ? recipeList[0].id : 1);
      chai.request(BASE_URL)
      .delete(DELETE_RECIPE_PATH + targetId)
      .end((err, res) => {
        res.should.have.status(200);
        res.body.should.be.a('object');
        res.body.should.have.property('message');
        done();
      });
    });
  });
});