##################
Gestion des règles
##################

La commande de gestion des :term:`règles<règle>` de  transfert est ``rule``.
Cette commande doit ensuite être suivie d'une action. La liste complète des
actions est disponible :any:`ici <reference-cli-client-rules>`.


Ajouter une règle
=================

Pour créer une règle, la commande est ``add``. Les options de commande
suivantes doivent être fournies:

- ``-n``: le nom de la règle
- ``-c``: un commentaire sur la règle (optionnel)
- ``-d``: la direction de transfert des fichiers, ``SEND`` ou ``RECEIVE``
- ``-p``: le chemin utilisé pour identifier la règle lorsque le protocole ne le
  permet pas
- ``-i``: le chemin de destination des fichiers (optionnel)
- ``-o``: le chemin source des fichiers (optionnel)
- ``-r``: une pré-tâche, peut être répété pour ajouter plusieurs tâches
- ``-s``: une post-tâche, peut être répété pour ajouter plusieurs tâches
- ``-e``: une tâche d'erreur, peut être répété pour ajouter plusieurs tâches

Les tâches doivent être données en format JSON. Voir :any:`reference-tasks`
pour plus de détails sur le format des tâches.

.. note::
   L'ordre dans lequel les tâches (pré/post/erreur) seront exécutées est le même
   que l'ordre dans lequel les tâches ont été spécifiées dans la commande de
   création.

**Exemple**

L'exemple suivant ajoute une règle de réception nommée 'rebond archive', identifiée
par le chemin '/rebond'. Cette règle possède 2 post-traitements. Le premier crée une
copie du fichier reçu dans le dossier '/archive', le deuxième programme un transfert
du fichier reçu vers le partenaire 'sshd'.

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' rule add -n 'rebond archive' -d 'RECEIVE' -p '/rebond' -s '{"type":"COPY","args":{"path": "/archive"}}' -s '{"type":"TRANSFER","args":{"file":"#TRUEFULLPATH#","to":"sshd","as":"toto","rule":"send"}}'


Modifier une règle
==================

Pour modifier une règle existante, la commande est ``rule update``. Cette commande
doit être suivie du nom de la règle à modifier. Les options de commandes sont
identiques à la commande ``add``. Il est possible d'omettre une ou plusieurs
options pour faire une mise à jour partielle.

.. warning::
   Il est impossible de faire une mise à jour partielle d'une chaine de
   traitements. Par conséquent, si vous souhaitez modifier une des chaines de
   traitement (pre/post/erreur), la chaine en question doit impérativement être
   rentrée en intégralité (y compris les tâches déjà rentrées précédemment).

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' rule update 'rebond archive' -p '/rebond' -i '/rebond/in'


Consulter les règles
====================

Pour lister les règles connues de la *gateway*, la commande est ``rule list``.
Les options de commande permettent de filtrer les résultats selon divers critères,
pour plus de détails, voir la :any:`documentation
<reference-cli-client-rules-list>` de la commande ``list``.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' rule list

Pour consulter une règle en particulier, la commande est ``get`` suivie du nom
de la règle.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' rule get 'rebond archive'


Supprimer une règle
===================

Pour supprimer une règle, la commande est ``rule delete``, suivie ensuite du nom
de la règle à supprimer.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' rule delete 'rebond archive'


Restreindre une règle
=====================

Par défaut, après ajout d'une règle, tous les serveurs, partenaires et comptes
(locaux et distants) peuvent utiliser cette règle. Il est cependant possible de
restreindre l'utilisation d'une règle pour que seuls certains puissent l'utiliser.

Chaque règle dispose d'une liste blanche, contenant la liste des différents agents
autorisés à utiliser la règle en question. Si cette liste est vide, alors la règle
est utilisable par tous.

.. note::

   Pour qu'un transfert puisse s'exécuter, il est nécessaire qu'au moins un des
   deux agents impliqués (serveur + compte local ou partenaire + compte distant
   suivant le sens de la connection) soit présent sur la liste blanche de la
   règle.

   Cela signifie donc qu'ajouter un serveur à la liste blanche d'une règle ajoute
   également *de facto* tous les comptes locaux rattachés à ce serveur. Idem pour
   les partenaires et les comptes distants.


Pour ajouter un agent à la liste blanche d'une règle, les commandes sont :

* ``server 'NOM' authorize 'RÈGLE'`` pour ajouter un serveur
* ``partner 'NOM' authorize 'RÈGLE'`` pour ajouter un partenaire
* ``account local 'LOGIN' authorize 'RÈGLE'`` pour ajouter un serveur
* ``account remote 'LOGIN' authorize 'RÈGLE'`` pour ajouter un serveur

Par exemple, la commande

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server 'WAARP SFTP' authorize 'send'

ajoute le serveur 'WAARP SFTP' ajoute le serveur local 'WAARP SFTP' à la liste
blanche de la règle 'send'.


Retirer un agent de la liste blanche se fait de manière similaire, la commande
``authorize`` doit juste être remplacée par la commande ``revoke``.

Par exemple, pour retirer le serveur 'WAARP SFTP' de la liste blanche, la commande
est :

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' server 'WAARP SFTP' revoke 'send'


Alternativement, il est possible d'effacer intégralement la liste blanche d'une
règle via la commande ``rule allow`` suivie du nom de la règle.

Par exemple, la commande suivante efface la liste blanche de la règle 'send',
rendant, de fait, la règle utilisable par tous :

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' rule allow 'send'
