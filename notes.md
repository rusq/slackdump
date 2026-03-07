Issue # 560
-----------

Convert from database to export.
Convert from export to database.
Run comparison.

Run convert of the tmp/reduced to database.
```
$ sd convert -f database tmp/reduced
```

in tmp/compare, start sqlite
run
```sql
	attach database 'conv.sqlite' as conv;
	attach database 'orig.sqlite' as orig;
```

Output of the missing data:
```sql
	with missing as (select id from orig.message where id not in (select id from conv.message))
	select id,chunk_id,channel_id,ts,parent_id,thread_ts,is_parent from orig.message where id in (select * from missing);
```

comparison of IDs:
```sql
select o.id,o.cnt,c.id,c.cnt from (select id,count(1) cnt from orig.message group by id) o left join (select id,count(1) cnt from conv.message group by id) c on o.id=c.id where o.cnt <> coalesce(c.cnt,-1);
```
