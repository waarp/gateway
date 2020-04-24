Consulter une règle
===================

.. http:get:: /api/rules/(string:rule_name)

   Renvoie la règle demandée.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: La règle a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La règle demandée n'existe pas


   :resjson string name: Le nom de la règle
   :resjson string comment: Un commentaire optionnel à propos de la règle (description...)
   :resjson bool isSend: Si vrai, la règle ne peut être utilisée que pour l'envoi
      de fichiers, si faux, la règle ne peut être utilisée que pour la réception
      de fichiers
   :resjson string path: Le chemin d'identification de la règle. Sert à identifier
      la règle lors d'un transfert si le protocole ne le permet pas. Doit être un
      chemin absolu.
   :resjson string inPath: Le dossier de destination de la règle. Tous les fichiers
      transférés avec cette règle sont envoyés dans ce dossier.
   :resjson string outPath: Le dossier source de la règle. Tous les fichiers
      transférés avec cette règle sont récupérés depuis ce dossier.
   :resjson array preTasks: La liste des pré-traitements de la règle.

      * **type** (*string*) - Le type de traitements.
      * **reception** (*object*) - Les arguments du traitement. La structure dépend du type de traitement.

   :resjson array postTasks: La liste des post-traitements de la règle.

      * **type** (*string*) - Le type de traitement.
      * **reception** (*object*) - Les arguments du traitement. La structure dépend du type de traitement.

   :resjson array errorTasks: La liste des traitements d'erreur de la règle.

      * **type** (*string*) - Le type de traitement.
      * **reception** (*object*) - Les arguments du traitement. La structure dépend du type de traitement.

   :resjson object authorized: Les agents autorisés à utiliser cette règle. Par
      défaut, si cet objet est vide, alors la règle peut être utilisée par tous
      le monde, sans exception.

      * **servers** (*array* of *string*) - La liste des serveurs locaux autorisés à utiliser la règle.
      * **partners** (*array* of *string*) - La liste des partenaires distants autorisés à utiliser la règle.
      * **localAccounts** (*object*) - La liste des comptes locaux autorisés à utiliser la règle. Chaque champ représente un serveur auquel on associe la liste des comptes qui lui sont affiliés.
      * **remoteAccounts** (*object*) - La liste des comptes locaux autorisés à utiliser la règle. Chaque champ représente un serveur auquel on associe la liste des comptes qui lui sont affiliés.


   .. admonition:: Exemple de requête

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/rules/règle_1 HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   .. admonition:: Exemple de réponse

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 1064

         {
           "name": "règle_1",
           "comment": "ceci est un exemple de règle d'envoi",
           "isSend": true,
           "path": "/chemin/de/la/règle",
           "outPath": "/chemin/source/des/fichiers",
           "inPath": "/chemin/destination/des/fichiers",
           "preTasks": [{
             "type": "COPY",
             "args": {"path":"/chemin/de/copie"}
           },{
             "type": "EXEC",
             "args": {"path":"/chemin/du/script","args":"{}","delay":"0"}
           }],
           "postTasks": [{
             "type": "DELETE",
             "args": {}
           },{
             "type": "TRANSFER",
             "args": {"file":"/chemin/du/fichier","to":"waarp_sftp","as":"toto","rule":"règle_2"}
           }],
           "errorTasks": [{
             "type": "MOVE",
             "args": {"path":"/chemin/de/déplacement"}
           },{
             "type": "RENAME",
             "args": {"path":"/chemin/du/renommage"}
           }],
           "authorized": {
             "servers": ["serveur_sftp", "serveur_http"],
             "partners": ["waarp_sftp", "waarp_r66"],
             "localAccounts": {
               "serveur_ftp": ["toto", "titi"],
               "serveur_r66": ["titi", "tata"]
             },
             "remoteAccounts": {
               "waarp_http": ["tata", "toto"],
               "waarp_ftp": ["tutu", "titi"]
             }
           }
         }