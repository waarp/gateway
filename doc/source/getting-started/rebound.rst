####################
Création d'un rebond
####################


Dans cette partie, nous allons réutiliser la configuration générée dans les
pages précédentes, et la modifier pour automatiser un rebond entre les
différents agents de transfert :

1. Un fichier va être envoyer à la Gateway en SFTP (utilisation de la première
   règle définie)
2. Le fichier reçu est renvoyé avec R66 vers le serveur Waarp R66 à l'aide de la
   deuxième règle définie.

Pour ce faire, nous allons modifier la première règle et ajouter un traitement
post-transfert pour ré-émettre le fichier automatiquement.


Modification de la règle ``sftp_recv``
======================================

Pour permettre d'automatiser le rebond, la chaîne de traitement post-transfert
doit avoir deux tâches :

1. La première déplace le fichier du dossier de réception de la première règle
   (en l'occurrence, le dossier ``in`` de la Gateway) vers le dossier d'envoi de
   la seconde règle (ici, le dossier ``out`` de la seconde règle). C'est une
   tâche :any:`reference-tasks-moverename`.
2. La deuxième tâche lance un transfert avec la seconde règle. C'est une tâche
   :any:`reference-tasks-transfer`.

Les tâches sont définies au format JSON comme un objet avec deux propriétés :
``type``, pour indiquer le type de la tâche, et ``args``, pour fournir les
arguments nécessaires à l'exécution de la tâche.

Dans notre cas, la tâche :any:`reference-tasks-moverename` a besoin d'un
argument ``path``, pour indiquer le nouveau chemin du fichier ; et la tâche
:any:`reference-tasks-transfer` a besoin *a minima* des mêmes arguments que ceux
à renseigner dans la commande de création d'un transfert, à savoir ``file``,
``to``, ``as`` et ``rule``.

Nous allons également utiliser des substitution dans les différentes tâches.
Celles-ci sont remplacées au moment de l'exécution de la tâche par des valeurs
dépendant du fichier transféré ou du transfert lui-même. Par exemple,
``#ORIGINALFILENAME#`` sera remplacé par le nom initial du fichier.

En mettant tout bout à bout, les traitements peuvent être ajoutés à la règle
avec la commande suivante :

.. code-block:: shell-session

   $ waarp-gateway rule update "sftp_recv" "receive" \
      --post '{"type": "MOVERENAME", "args": {"path":"#OUTPATH#/#ORIGINALFILENAME#"}}' \
      --post '{"type": "TRANSFER", "args":{"file": "#OUTPATH#/#ORIGINALFILENAME#", "using":"r66_client" "to":"r66_server", "as":"gw_r66user", "rule":"default"}}'
   The rule sftp_recv was successfully updated.


.. seealso::

   La liste des tâches disponibles et leurs arguments est documentée
   :any:`ici <reference-tasks>`.

   La liste des substitutions est consultable :any:`ici <reference-tasks-substitutions>`


Test de transfert
=================

Nous pouvons maintenant effectuer un transfert de test. Pour cela, nous allons
déposer un fichier en SFTP sur la Gateway, et vérifier que le rebond a bien été
pris en compte et que deux transferts ont été faits :

.. code-block:: shell-session

   $ sftp -P 2223 myuser@localhost
   myuser@localhost's password: 
   Connected to myuser@localhost.
   sftp> put test.txt sftp_recv/test02.txt
   Uploading test.txt to /sftp_recv/test02.txt
   test.txt                                                                                              100%   20     5.7KB/s   00:00    
   sftp> quit

Après avoir établi une connexion avec la Gateway, nous avons déposé un fichier
avec la commande ``put`` dans le dossier ``sftp_recv`` que nous avons défini
ci-dessus comme le ``path`` de la règle ``sftp_recv``.

Nous pouvons vérifier que les transfert se bien passés dans l'historique des
transferts de la Gateway :

.. code-block:: shell-session

   $ waarp-gateway transfer list
   Transfers:
   [...]
   ● Transfer 25 (as server) [DONE]
       Way:             receive
       Protocol:        sftp
       Rule:            sftp_recv
       Requester:       myuser
       Requested:       sftp_server
       Local filepath:  /etc/waarp-gateway/out/test04.txt
       Remote filepath: /test04.txt
       Start date:      2020-10-02T15:10:48Z
       End date:        2020-10-02T15:10:49Z
   ● Transfer 26 (as client) [DONE]
       Way:             send
       Protocol:        r66
       Rule:            default
       Requester:       gw_r66user
       Requested:       r66_server
       Local filepath:  /etc/waarp-gateway/out/test04.txt
       Remote filepath: /test04.txt
       Start date:      2020-10-02T15:10:49Z
       End date:        2020-10-02T15:10:49Z
   
Le fichier disponible est maintenant dans le dossier ``in`` de la Gateway.
Comme nous n'avons pas spécifié de dossier spécifique dans la règle
``sftp_send``, c'est le dossier par défaut du service qui est utilisé :

.. code-block:: shell-session

   # s -l /home/sftpuser/
   total 8
   -rw-rw-r--. 1 sftpuser sftpuser 13 Sep 17 17:27 a-envoyer.txt
   -rw-rw-r--. 1 sftpuser sftpuser 20 Oct  2 15:10 test04.txt




