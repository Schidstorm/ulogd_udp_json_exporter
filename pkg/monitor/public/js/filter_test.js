import * as filter from './filter.js';
import util from 'util';

runTests();

function test() {
    const { token, rpn } = filter.parse("field1 == 100 and ( field2 < 200 or field3 >= 300 )");
    console.log("Tokens:");
    console.log(util.inspect(token, { depth: null, colors: true }));
    console.log("RPN:");
    console.log(util.inspect(rpn, { depth: null, colors: true }));
}

// runTests();

function runTests() {
    const tests = [
        testParseAritOperator,
        testParseSize,
        testParseNumber,
        testParseLabel,
        testParseBoolOperator,
        testParseEquation,
        testExpression,
    ];

    for (const test of tests) {
        console.log(`Running ${test.name}...`);
        test();
    }

    console.log("All tests passed!");
}

function testExpression() {
    testCases(new filter.ExpressionParser(), [
        [
            "( field1 == 100 ) extra", 
            {
                "type":"expression",
                "value":[
                    {
                        "type":"parenthesized",
                        "value":[
                            {
                                "type":"expression",
                                "value":[
                                    {
                                        "value":[
                                            {"type":"label","value":"field1","begin":2,"length":6},
                                            {"type":"arit-operator","value":"==","begin":9,"length":2},
                                            {"type":"number","value":"100","begin":12,"length":3}
                                        ],"begin":2,"length":13
                                    }
                                ],
                                "begin":2,
                                "length":14
                            }
                        ],
                        "begin":0,
                        "length":17
                    }
                ],
                "begin":0,
                "length":18
            }, 
            'Test 1'
        ],
        [
            "field_name<200", 
            {
                "type":"expression",
                "value":[
                    {
                        "value":[
                            {"type":"label","value":"field_name","begin":0,"length":10},
                            {"type":"arit-operator","value":"<","begin":10,"length":1},
                            {"type":"number","value":"200","begin":11,"length":3}
                        ],
                        "begin":0,
                        "length":14
                    }
                ],
                "begin":0,
                "length":14
            }, 
            'Test 2'
        ],
        ["( invalid equation )",{"error":"Expected end of input"}, 'Test 3'],
        [
            "field1 == 100 and field2 < 200", 
            {
                "type":"expression",
                "value":[
                    {
                        "value":[
                            {"type":"label","value":"field1","begin":0,"length":6},
                            {"type":"arit-operator","value":"==","begin":7,"length":2},
                            {"type":"number","value":"100","begin":10,"length":3}
                        ],
                        "begin":0,
                        "length":13
                    },
                    {
                        "type":"bool-operator",
                        "value":"and",
                        "begin":14,
                        "length":3
                    },
                    {
                        "value":[
                            {"type":"label","value":"field2","begin":18,"length":6},
                            {"type":"arit-operator","value":"<","begin":25,"length":1},
                            {"type":"number","value":"200","begin":27,"length":3}
                        ],
                        "begin":18,
                        "length":12
                    }
                ],
                "begin":0,
                "length":30
            }, 'Test 4'],
    ]);
}

function testParseEquation() {
    testCases(new filter.EquationParser(), [
        ["field1 == 100", {
            value:[
                {"type":"label","value":"field1","begin":0,"length":6},
                {"type":"arit-operator","value":"==","begin":7,"length":2},
                {"type":"number","value":"100","begin":10,"length":3}
            ],"begin":0,"length":13}, 'Test 1'],
        ["field_name<200", {
            value:[
                {"type":"label","value":"field_name","begin":0,"length":10},
                {"type":"arit-operator","value":"<","begin":10,"length":1},
                {"type":"number","value":"200","begin":11,"length":3}
            ],"begin":0,"length":14}, 'Test 2'],
        ["invalid equation", {"error":"Expected pattern /^(==|=|>|<)/, got 'equation...'"}, 'Test 3' ],
    ]);
}

function testParseLabel() {
    testCases(new filter.LabelParser(), [
        ["field_name123", { "type": "label", "value": "field_name123", "begin": 0, "length": 13 }, 'Test 1'],
        ["_startValue", { "type": "label", "value": "_startValue", "begin": 0, "length": 11 }, 'Test 2'],
        ["9invalidStart", { "error": "Expected pattern /^[a-zA-Z_][a-zA-Z0-9_-]*/, got '9invalidSt...'" }, 'Test 3'],
        ["valid-name-456", { "type": "label", "value": "valid-name-456", "begin": 0, "length": 14 }, 'Test 4'],
    ]);
}

function testParseBoolOperator() {
    testCases(new filter.BoolOperatorParser(), [
        ["and", { "type": "bool-operator", "value": "and", "begin": 0, "length": 3 }, 'Test 1'],
        ["or", { "type": "bool-operator", "value": "or", "begin": 0, "length": 2 }, 'Test 2'],
    ]);
}

function testParseAritOperator() {
    testCases(new filter.AritOperatorParser(), [
        ["==", { type: 'arit-operator', value: '==', begin: 0, length: 2 }, 'Test 1'],
        ["= ", { type: 'arit-operator', value: '==', begin: 0, length: 1 }, 'Test 2'],
        [">", { type: 'arit-operator', value: '>', begin: 0, length: 1 }, 'Test 3'],
        ["<", { type: 'arit-operator', value: '<', begin: 0, length: 1 }, 'Test 4'],
    ]);
}

function testParseSize() {
    testCases(new filter.SizeParser(), [
        ["123b", { "type": "size", "value": "123", "begin": 0, "length": 4 }, 'Test 1'],
        ["456kb", { "type": "size", "value": "456000", "begin": 0, "length": 5 }, 'Test 2'],
        ["7mb", { "type": "size", "value": "7000000", "begin": 0, "length": 3 }, 'Test 3'],
        ["8gib", { "type": "size", "value": "8589934592", "begin": 0, "length": 4 }, 'Test 4'],
        ["9tb", { "type": "size", "value": "9000000000000", "begin": 0, "length": 3 }, 'Test 5'],
        ["no_size", { "error": "Expected size number" }, 'Test 6'],
    ]);
}

function testParseNumber() {
    testCases(new filter.NumberParser(), [
        ["12345", { "type": "number", "value": "12345", "begin": 0, "length": 5 }, 'Test 1'],
        ["00123", { "type": "number", "value": "123", "begin": 0, "length": 5 }, 'Test 2'],
        ["42", { "type": "number", "value": "42", "begin": 0, "length": 2 }, 'Test 4'],
        ["7", { "type": "number", "value": "7", "begin": 0, "length": 1 }, 'Test 5'],
        ["no_digits_here", { "error": "Expected number" }, 'Test 3'],
    ]);
}

function testCases(parser, cases) {
    for (const item of cases) {
        testCase.apply(null, [parser].concat(item));
    }
}

function testCase(parser, data, expectedValue, testName) {
    let result;
    let cachedError = null;
    try {
        result = filter.parse(data, parser).token;
    } catch (e) {
        result = { error: e.message };
        cachedError = e;
    }
    const success = JSON.stringify(result) === JSON.stringify(expectedValue);
    if (!success && cachedError) {
        console.error(cachedError);
    }
    console.assert(success, `Failed ${testName}: expected value ${JSON.stringify(expectedValue)}, got ${JSON.stringify(result)}`);
}