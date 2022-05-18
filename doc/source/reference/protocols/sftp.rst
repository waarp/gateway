.. _ref-proto-sftp:

====
SFTP
====

Bien que le protocole comporte de nombreuses fonctionnalités, seules celles
pertinentes pour le MFT ont été implémentées dans la *gateway*.

Il est donc possible d'initier un transfert via les commandes ``Put`` ou ``Get``
(suivant le sens du transfert). Dans les 2 cas, étant donné que SFTP n'offre pas
de mécanisme pour transmettre le nom de la règle à utiliser, c'est donc le chemin
du fichier qui est utilisé pour déterminer la règle.

L'implémentation de SFTP dans la *gateway* supporte également les commandes
``List`` et `Stat` permettant de récupérer les fichiers disponibles sur le serveur.
Cependant, l'implémentation dans la *gateway* diffère des implémentations classiques,
car elle masque les dossiers réels se trouvant sous la racine du serveur. À la
place, le serveur donne une liste de dossiers correspondants aux `path` de toutes
les règles utilisables par l'utilisateur.

Lister le contenu d'un de ces dossiers affichera la liste des fichiers pouvant
être récupéré avec la règle correspondante. Par conséquent, les dossiers des
règles de réception seront donc toujours vides (à moins qu'il existe une règle
d'envoi ayant le même `path`). Pour résumer, cela signifie que les fichiers
déposés sur le serveur ne seront pas visibles une fois le transfert terminé.

|

Toutes les autres commandes SFTP, à savoir ``Setstat``, ``Rename``, ``Rmdir``,
``Mkdir``, ``Link``, ``Symlink``, ``Remove`` & ``Readlink`` ne sont pas
implémentées, car non-pertinentes pour le MFT.