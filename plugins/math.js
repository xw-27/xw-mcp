module.exports = [
    {
        name: 'add',
        description: '加法运算',
        inputSchema: {
            type: 'object',
            properties: {
                a: { type: 'number' },
                b: { type: 'number' }
            }
        },
        execute: function(params) {
            return { result: params.a + params.b };
        }
    },
    {
        name: 'subtract',
        description: '减法运算',
        inputSchema: {
            type: 'object',
            properties: {
                a: { type: 'number' },
                b: { type: 'number' }
            }
        },
        execute: function(params) {
            return { result: params.a - params.b };
        }
    },
    {
        name: 'multiply',
        description: '乘法运算',
        inputSchema: {
            type: 'object',
            properties: {
                a: { type: 'number' },
                b: { type: 'number' }
            }
        },
        execute: function(params) {
            return { result: params.a * params.b };
        }
    },
    {
        name: 'divide',
        description: '除法运算',
        inputSchema: {
            type: 'object',
            properties: {
                a: { type: 'number' },
                b: { type: 'number' }
            }
        },
        execute: function(params) {
            if (params.b === 0) {
                return { error: '除数不能为0' };
            }
            return { result: params.a / params.b };
        }
    },
    {
        name: 'power',
        description: '幂运算',
        inputSchema: {
            type: 'object',
            properties: {
                base: { type: 'number' },
                exponent: { type: 'number' }
            }
        },
        execute: function(params) {
            return { result: Math.pow(params.base, params.exponent) };
        }
    },
    {
        name: 'sqrt',
        description: '平方根',
        inputSchema: {
            type: 'object',
            properties: {
                value: { type: 'number' }
            }
        },
        execute: function(params) {
            if (params.value < 0) {
                return { error: '负数不能开平方根' };
            }
            return { result: Math.sqrt(params.value) };
        }
    },
    {
        name: 'abs',
        description: '绝对值',
        inputSchema: {
            type: 'object',
            properties: {
                value: { type: 'number' }
            }
        },
        execute: function(params) {
            return { result: Math.abs(params.value) };
        }
    },
    {
        name: 'max',
        description: '最大值',
        inputSchema: {
            type: 'object',
            properties: {
                values: { type: 'array', items: { type: 'number' } }
            }
        },
        execute: function(params) {
            return { result: Math.max.apply(null, params.values) };
        }
    },
    {
        name: 'min',
        description: '最小值',
        inputSchema: {
            type: 'object',
            properties: {
                values: { type: 'array', items: { type: 'number' } }
            }
        },
        execute: function(params) {
            return { result: Math.min.apply(null, params.values) };
        }
    },
    {
        name: 'round',
        description: '四舍五入',
        inputSchema: {
            type: 'object',
            properties: {
                value: { type: 'number' },
                decimals: { type: 'number' }
            }
        },
        execute: function(params) {
            var multiplier = Math.pow(10, params.decimals || 0);
            return { result: Math.round(params.value * multiplier) / multiplier };
        }
    }
];