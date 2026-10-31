const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const root = path.join(__dirname, '../../internal/web/templates/static/js');
const videoSource = fs.readFileSync(path.join(root, 'videogen.js'), 'utf8');

function player(navigator) {
  const calls = {contexts:0, sources:0, connections:0, resumes:0};
  class AudioContext {
    constructor() { calls.contexts++; this.state = 'suspended'; this.destination = {}; }
    createAnalyser() { return {connect(){calls.connections++;}}; }
    createMediaElementSource() { calls.sources++; return {connect(){calls.connections++;}}; }
    resume() { calls.resumes++; this.state='running'; return Promise.resolve(); }
  }
  const window = {ap:{audio:{}}, AudioContext};
  vm.runInNewContext(videoSource, {window, navigator, console});
  return {video:window.VideoGen, calls};
}
for (const [name,navigator] of [
  ['iPhone Safari',{userAgent:'Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X)',platform:'iPhone',maxTouchPoints:5}],
  ['iPad desktop mode',{userAgent:'Mozilla/5.0 (Macintosh; Intel Mac OS X)',platform:'MacIntel',maxTouchPoints:5}],
  ['iPhone standalone',{userAgent:'Mozilla/5.0 (iPhone)',platform:'iPhone',maxTouchPoints:5,standalone:true}],
]) test(name+' never connects playback to a suspendable audio graph',()=>{
  const {video,calls}=player(navigator);
  // Covers page prewarming, user gestures, play events and opening details.
  video.resumeAudioRouting(); video.initAudioContext(); video.connectAudioSource();
  video.isLocalAudio=true; video.localAudio={}; video.connectAudioSource();
  assert.deepEqual(calls,{contexts:0,sources:0,connections:0,resumes:0});
});
test('desktop routing remains connected once and resumes on interaction',()=>{
  const {video,calls}=player({userAgent:'Macintosh',platform:'MacIntel',maxTouchPoints:0});
  video.resumeAudioRouting(); video.initAudioContext(); video.connectAudioSource();
  assert.deepEqual(calls,{contexts:1,sources:1,connections:2,resumes:1});
});
const appSource=fs.readFileSync(path.join(root,'app.js'),'utf8');
const sessionSource=appSource.slice(appSource.indexOf('function configureAudioPlaybackSession()'),appSource.indexOf('function resumeVideoGenAudioRouting()'));
test('playback session is feature-detected and tolerates unsupported setters',()=>{
  for(const navigator of [{},{audioSession:{type:'auto'}},{audioSession:{get type(){return 'auto'},set type(_){throw new Error('unsupported')}}}]) {
    const context=vm.createContext({navigator});
    vm.runInContext(sessionSource+'\nconfigureAudioPlaybackSession();',context);
    if(navigator.audioSession && Object.getOwnPropertyDescriptor(navigator.audioSession,'type').value) assert.equal(navigator.audioSession.type,'playback');
  }
});
