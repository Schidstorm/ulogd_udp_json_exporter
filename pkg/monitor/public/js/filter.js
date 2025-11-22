
export class Filter {
    constructor() {
        this._eventElement = document.createElement('div');
        this.expressionString = "";
        this.token = null;
        this.rpn = [];
    }

    addEventListener(type, listener) {
        this._eventElement.addEventListener(type, listener);
    }

    removeEventListener(type, listener) {
        this._eventElement.removeEventListener(type, listener);
    }

    dispatchEvent(event) {
        this._eventElement.dispatchEvent(event);
    }

    setExpression(expressionString) {
        this.expressionString = expressionString;
        
        try {
            const {token, rpn} = parse(expressionString);
            this.token = token;
            this.rpn = rpn;
        } catch (e) {
            this.dispatchEvent(new CustomEvent('change', { detail: { valid: false } }));
            return;
        }

        this.dispatchEvent(new CustomEvent('change', { detail: { 
            valid: true,
            rpn: this.rpn,
            token: this.token,
        }}));
    }

    getTokensAt(index) {
        let possibilities = [];
        this._walkToken((token) => {
            if (token.begin <= index && (token.begin + token.length) >= index) {
                possibilities.push(token);
            }
        });

        return possibilities.sort((a, b) => a.length - b.length);
    }

    _walkToken(callback) {
        function walk(node) {
            callback(node);
            if (Array.isArray(node.value)) {
                for (const child of node.value) {
                    walk(child);
                }
            }
        }
        walk(this.token);
    }
}

export function parse(data, parser=new ExpressionParser()) {
    const parsed = parser.parse(data, 0);
    if (!parsed.ok || parsed.length !== data.length) {
        if (parsed.error) {
            throw new Error(parsed.error);
        }
    }

    const token = removeUnimportantTokens(parsed);
    const rpn = buildRPN(token);
    return new ParseResult(token, rpn);
}

class ParseResult {
    constructor(token, rpn) {
        this.token = token;
        this.rpn = rpn;
    }
}


function removeUnimportantTokens(token) {
    switch (token.parser.name) {
        case 'space':
        case 'parenthesis-open':
        case 'parenthesis-close':
            return undefined;
        case 'repeat':
            if (token.value.length === 0) {
                return undefined;
            }
    }

    let value = token.value;
    if (Array.isArray(value)) {
        value = token.value.map(childToken => removeUnimportantTokens(childToken)).filter(v => v !== undefined);
    }
    return { type: token.parser.name, value: value, begin: token.begin, length: token.length };
}


function buildRPN(token) {
    function addParenthesized(node) {
        if (node.type === 'expression' && node.value.length > 3) {
            for (const operator of BoolOperatorParser.operators()) {
                const operatorIndeces = node.value
                    .map((v, i) => ({ v, i }))
                    .filter(vi => vi.v.type === 'bool-operator' && vi.v.value === operator)
                    .map(vi => vi.i);

                if (operatorIndeces.length === 0) {
                    continue;
                }

                const opIndex = operatorIndeces[0];
                
                const finalSubExpression = {
                    type: 'expression',
                    value: [
                        node.value[opIndex - 1],
                        node.value[opIndex],
                        node.value[opIndex + 1],
                    ],
                };

                node.value.splice(opIndex - 1, 3, finalSubExpression);
                break;
            }

            node = addParenthesized(node);
        }

        if (Array.isArray(node.value)) {
            for (let i = 0; i < node.value.length; i++) {
                node.value[i] = addParenthesized(node.value[i]);
            }
        }
        return node;
    }

    const result = [];
    function convertToRpm(node) {
        if (Array.isArray(node.value)) {
            for (const child of node.value) {
                convertToRpm(child);
            }
        }
        switch (node.type) {
        case 'equation':
            result.push({
                type: 'arit-operator',
                value: node.value.filter(v => v.type === 'arit-operator')[0].value,
            });
            break;
        case 'expression':
            if (node.value.length < 3) {
                break;
            }
            result.push({
                type: 'bool-operator',
                value: node.value.filter(v => v.type === 'bool-operator')[0].value,
            });
            break;
        case 'parenthesized':
        case 'arit-operator':
        case 'bool-operator':
            // do nothing
            break;
        default:
            result.push({
                type: node.type,
                value: node.value,
            });
        }
    }

    convertToRpm(addParenthesized(token));
    return result;
}

export class Parser {
    constructor(name) {
        this.name = name;
    }

    parse(data, index) { console.error("Parser.parse not implemented"); }
    [Symbol.toStringTag]() {
        return this.name;
    }

    [Symbol.for('nodejs.util.inspect.custom')]() {
        return this.name
    }
}

export class OrParser extends Parser {
    constructor(parsers, name="or") {
        super(name);
        this.parsers = parsers;
    }

    parse(data, index) {
        for (let parser of this.parsers) {
            const result = parser.parse(data, index);
            if (result.ok) {
                return result;
            }
        }

        const expectedTokens = this.parsers.map(p => p.name).join(", ");
        return Token.fail(this, `Expected one of: ${expectedTokens}, got '${data.substring(index, index+10)}...'`);
    }
}

export class RegexParser extends Parser {
    constructor(regex, name="regex") {
        super(name);
        this.regex = regex;
    }

    parse(data, index) {
        const match = this.regex.exec(data.substring(index));
        if (match && match.index === 0) {
            return Token.ok(this, match[0].length, match[0]).withPos(index);
        } else {
            return Token.fail(this, `Expected pattern ${this.regex}, got '${data.substring(index, index+10)}...'`);
        }
    }
}

export class ConcatParser extends Parser {
    constructor(parsers, name="concat") {
        super(name);
        this.parsers = parsers;
    }

    parse(data, index) {
        let totalLength = 0;
        const values = [];

        for (let parser of this.parsers) {
            const result = parser.parse(data, index + totalLength);
            if (!result.ok) {
                return result;
            }
            totalLength += result.length;
            values.push(result);
        }

        return Token.ok(this, totalLength, values).withPos(index);
    }

    [Symbol.for('nodejs.util.inspect.custom')]() {
        return `${this.name}(${this.parsers.map(p => p.name).join(", ")})`;
    }
}

export class EquationParser {
    constructor(name="equation") {
        this.valueParser = new OrParser([
            new Delayed(() => new ValueParser()),
            new Delayed(() => new LabelParser()),
            new Delayed(() => new LabelPlaceholderParser())
        ]);
        this.spaceParser = new SpaceParser();
        this.aritOperatorParser = new AritOperatorParser();
    }

    parse(data, index) {
        const beginIndex = index;

        const leftSpaceResult = this.spaceParser.parse(data, index);
        index += leftSpaceResult.length;

        const leftValueResult = this.valueParser.parse(data, index);
        if (!leftValueResult.ok) {
            return leftValueResult;
        }
        index += leftValueResult.length;

        const middleSpaceResult = this.spaceParser.parse(data, index);
        index += middleSpaceResult.length;

        const operatorResult = this.aritOperatorParser.parse(data, index);
        if (!operatorResult.ok) {
            return operatorResult;
        }
        index += operatorResult.length;

        const rightSpaceResult = this.spaceParser.parse(data, index);
        index += rightSpaceResult.length;

        const rightValueResult = this.valueParser.parse(data, index);
        if (!rightValueResult.ok) {
            return rightValueResult;
        }
        index += rightValueResult.length;

        const totalLength = index - beginIndex;
        const value = [
            leftValueResult,
            operatorResult,
            rightValueResult
        ];

        return Token.ok(this, totalLength, value).withPos(beginIndex);
    }
}

export class ParenthesizeParser extends ConcatParser {
    constructor(name="parenthesized") {
        super([
            new Delayed(() => new ParenthesisOpenParser()),
            new Delayed(() => new SpaceParser()),
            new Delayed(() => new ExpressionParser()),
            new Delayed(() => new SpaceParser()),
            new Delayed(() => new ParenthesisCloseParser())
        ], name);
    }
}

export class ParenthesisOpenParser extends RegexParser {
    constructor(name="parenthesis-open") {
        super(/^\(/, name);
    }
}

export class ParenthesisCloseParser extends RegexParser {
    constructor(name="parenthesis-close") {
        super(/^\)/, name);
    }
}

export class ValueParser extends OrParser {
    constructor(name="value") {
        super([
            new Delayed(() => new SizeParser()),
            new Delayed(() => new NumberParser())
        ], name);
    }
}

export class ExpressionParser extends Parser {
    constructor(name="expression") {
        super(name);

        this.sub = new OrParser([
            new Delayed(() => new ParenthesizeParser()),
            new Delayed(() => new EquationParser()),
        ]);
        this.bool = new BoolOperatorParser();
        this.space = new SpaceParser();
    }

    parse(data, index) {
        const beginIndex = index;
        const results = [];
        let i = 0;

        while (true) {
            if (i > 0) {
                const spaceResult = this.space.parse(data, index);
                index += spaceResult.length;

                const boolResult = this.bool.parse(data, index);
                if (!boolResult.ok) {
                    break;
                }
                results.push(boolResult);
                index += boolResult.length;
            }

            const spaceResult = this.space.parse(data, index);
            index += spaceResult.length;

            const subResult = this.sub.parse(data, index);
            if (!subResult.ok) {
                break;
            }
            results.push(subResult);
            index += subResult.length;
            i++;
        }

        const eofResult = (new EofParser()).parse(data, index);
        if (!eofResult.ok && results.length === 0) {
            return eofResult;
        }

        return Token.ok(this, index - beginIndex, results).withPos(beginIndex);
    }
}

export class BoolOperatorParser extends Parser {
    constructor(name="bool-operator") {
        super(name);
    }

    static operators() {
        return ['and', 'or'];
    }

    parse(data, index) {
        const labelResult = (new LabelParser()).parse(data, index);
        if (!labelResult.ok) {
            return Token.fail(this, "Expected boolean operator");
        }

        const value = labelResult.value.toLowerCase();
        if(BoolOperatorParser.operators().includes(value)) {
            return Token.ok(this, labelResult.length, value).withPos(index);
        } else {
            return Token.fail(this, `Expected boolean operator, got '${value}'`);
        }
    }
}

export class AritOperatorParser extends RegexParser {
    constructor(name="arit-operator") {
        super(/^(==|=|>|<)/, name);
    }

    parse(data, index) {
        const result = super.parse(data, index);
        if (!result.ok) {
            return result;
        }

        let value = result.value;
        if (value === '=') {
            value = '==';
        }

        return Token.ok(this, result.length, value).withPos(index);
    }
}

export class LabelParser extends RegexParser {
    constructor(name="label") {
        super(/^[a-zA-Z_][a-zA-Z0-9_-]*/, name);
    }
}

export class LabelPlaceholderParser extends Parser {
    constructor(name="label-placeholder") {
        super(name);
    }

    parse(data, index) {
        return Token.ok(this, 0, null).withPos(index);
    }
}

export class SizeParser extends Parser {
    constructor(name="size") {
        super(name);
    }

    parse(data, index) {
        const originalIndex = index;
        const one = BigInt(1);
        const thousand = BigInt(1000);
        const oneKib = BigInt(1024);
        const sizeUnits = {
            'b': one,
            'kb': thousand,
            'mb': thousand * thousand,
            'gb': thousand * thousand * thousand,
            'tb': thousand * thousand * thousand * thousand,
            'kib': oneKib,
            'mib': oneKib * oneKib,
            'gib': oneKib * oneKib * oneKib,
            'tib': oneKib * oneKib * oneKib * oneKib
        };

        const numberResult = (new NumberParser()).parse(data, index);
        if (!numberResult.ok) {
            return Token.fail(this, "Expected size number");
        }

        index += numberResult.length;
        const unitRegex = /^(b|kb|mb|gb|tb|kib|mib|gib|tib)/i;
        const unitResult = (new RegexParser(unitRegex)).parse(data, index);
        if (!unitResult.ok) {
            return Token.fail(this, "Expected size unit");
        }

        const unit = unitResult.value.toLowerCase();
        const multiplier = sizeUnits[unit];
        const sizeValue = BigInt(numberResult.value) * multiplier;

        return Token.ok(this, numberResult.length + unitResult.length, sizeValue.toString(10)).withPos(originalIndex);
    }
}

export class NumberParser extends Parser {
    constructor(name="number") {
        super(name);
    }

    parse(data, index) {
        const numberRegex = /^[0-9]+/;
        const r = (new RegexParser(numberRegex)).parse(data, index);
        if (!r.ok) {
            return Token.fail(this, "Expected number");
        }

        return Token.ok(this, r.length, parseInt(r.value, 10)+'').withPos(index);
    }
}

export class Token {
    constructor(parser, length, value, ok, error=null) {
        this.parser = parser;
        this.length = length;
        this.value = value;
        this.ok = ok;
        this.begin = null;
        this.error = error;
    }

    static ok(parser, length, value) {
        return new Token(parser, length, value, true);
    }

    static fail(parser, reason=null) {
        return new Token(parser, 0, null, false, reason);
    }

    withPos(begin) {
        this.begin = begin;
        return this;
    }
}

export class SpaceParser extends Parser {
    constructor(name="space") {
        super(name);
    }

    parse(data, index) {
        const originalIndex = index;
        const space = ' '.charCodeAt(0);
        const tab = '\t'.charCodeAt(0);
        const newline = '\n'.charCodeAt(0);
        let length = 0;
        let result = '';

        while (index < data.length) {
            const charCode = data.charCodeAt(index);
            if (charCode === space || charCode === tab || charCode === newline) {
                index++;
                length++;
                result += data.charAt(index - 1);
            } else {
                break;
            }
        }

        return Token.ok(this, length, result).withPos(originalIndex);
    }
}

export class Delayed extends Parser {
    constructor(parserFactory) {
        super();
        this.parserFactory = parserFactory;
        this.parser = null;
    }

    getParser() {
        if (this.parser === null) {
            this.parser = this.parserFactory();
            this.name = this.parser.name;
        }
        return this.parser;
    }

    parse(data, index) {
        return this.getParser().parse(data, index);
    }

    [Symbol.for('nodejs.util.inspect.custom')]() {
        return this.getParser()[Symbol.for('nodejs.util.inspect.custom')]();
    }
}

export class EofParser extends Parser {
    constructor(name="eof") {
        super(name);
    }

    parse(data, index) {
        if (index >= data.length) {
            return Token.ok(this, 0, null).withPos(index);
        } else {
            return Token.fail(this, "Expected end of input");
        }
    }
}

