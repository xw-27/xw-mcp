module.exports = [
    {
        name: 'add',
        description: '两个数字相加',
        inputSchema: {
            type: 'object',
            properties: {
                augend: {
                    type: 'number',
                    description: '被加数'
                },
                addend: {
                    type: 'number',
                    description: '加数'
                }
            },
            required: ['augend', 'addend']
        },
        execute: function(params) {
            return { result: params.augend + params.addend };
        }
    },
    {
        name: 'subtract',
        description: '两个数字相减',
        inputSchema: {
            type: 'object',
            properties: {
                minuend: {
                    type: 'number',
                    description: '被减数'
                },
                subtrahend: {
                    type: 'number',
                    description: '减数'
                }
            },
            required: ['minuend', 'subtrahend']
        },
        execute: function(params) {
            return { result: params.minuend - params.subtrahend };
        }
    },
    {
        name: 'multiply',
        description: '两个数字相乘',
        inputSchema: {
            type: 'object',
            properties: {
                multiplicand: {
                    type: 'number',
                    description: '被乘数'
                },
                multiplier: {
                    type: 'number',
                    description: '乘数'
                }
            },
            required: ['multiplicand', 'multiplier']
        },
        execute: function(params) {
            return { result: params.multiplicand * params.multiplier };
        }
    },
    {
        name: 'divide',
        description: '两个数字相除',
        inputSchema: {
            type: 'object',
            properties: {
                dividend: {
                    type: 'number',
                    description: '被除数'
                },
                divisor: {
                    type: 'number',
                    description: '除数，不能为0'
                }
            },
            required: ['dividend', 'divisor']
        },
        execute: function(params) {
            if (params.divisor === 0) {
                return { error: '除数不能为0' };
            }
            return { result: params.dividend / params.divisor };
        }
    },
    {
        name: 'power',
        description: '计算幂运算',
        inputSchema: {
            type: 'object',
            properties: {
                base: {
                    type: 'number',
                    description: '底数'
                },
                exponent: {
                    type: 'number',
                    description: '指数'
                }
            },
            required: ['base', 'exponent']
        },
        execute: function(params) {
            return { result: Math.pow(params.base, params.exponent) };
        }
    },
    {
        name: 'sqrt',
        description: '计算平方根',
        inputSchema: {
            type: 'object',
            properties: {
                value: {
                    type: 'number',
                    description: '被开方数，不能为负数'
                }
            },
            required: ['value']
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
        description: '计算绝对值',
        inputSchema: {
            type: 'object',
            properties: {
                value: {
                    type: 'number',
                    description: '待求绝对值的数字'
                }
            },
            required: ['value']
        },
        execute: function(params) {
            return { result: Math.abs(params.value) };
        }
    },
    {
        name: 'max',
        description: '返回一组数字中的最大值',
        inputSchema: {
            type: 'object',
            properties: {
                values: {
                    type: 'array',
                    items: {
                        type: 'number',
                        description: '数字'
                    },
                    description: '数字数组'
                }
            },
            required: ['values']
        },
        execute: function(params) {
            return { result: Math.max.apply(null, params.values) };
        }
    },
    {
        name: 'min',
        description: '返回一组数字中的最小值',
        inputSchema: {
            type: 'object',
            properties: {
                values: {
                    type: 'array',
                    items: {
                        type: 'number',
                        description: '数字'
                    },
                    description: '数字数组'
                }
            },
            required: ['values']
        },
        execute: function(params) {
            return { result: Math.min.apply(null, params.values) };
        }
    },
    {
        name: 'round',
        description: '对数字进行四舍五入',
        inputSchema: {
            type: 'object',
            properties: {
                value: {
                    type: 'number',
                    description: '待四舍五入的数字'
                },
                decimals: {
                    type: 'number',
                    description: '保留小数位数，默认为0'
                }
            },
            required: ['value']
        },
        execute: function(params) {
            var multiplier = Math.pow(10, params.decimals || 0);
            return { result: Math.round(params.value * multiplier) / multiplier };
        }
    }
];