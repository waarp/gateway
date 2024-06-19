Programmer un transfert
=======================

.. http:post:: /api/transfers

   .. deprecated:: 0.5.0

      Les propriétés ``sourcePath`` et ``destPath`` de la requête ont été
      remplacées par les propriétés ``localFilepath`` et ``remoteFilepath``.

   .. deprecated:: 0.5.0

      La propriété ``startDate`` de la requête a été remplacée par la propriété
      ``start``.

   Programme un nouveau transfert avec les informations renseignées en format JSON dans
   le corps de la requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson bool isServer: Précise si la gateway était à l'origine du transfert
   :reqjson string rule: L'identifiant de la règle utilisée
   :reqjson bool isSend: Indique le transfert est un envoi (``true``) ou une
     réception (``false``).
   :reqjson string client: Le nom du client avec lequel effectuer le transfert.
     Peut être omit si la gateway ne possède qu'un seul client du protocole concerné,
     auquel cas, le client en question sera sélectionné automatiquement.
   :reqjson string account: Le nom du compte ayant demandé le transfert
   :reqjson string partner: Le nom du serveur/partenaire auquel le transfert a été demandé
   :reqjson string partner: Le nom du serveur/partenaire auquel le transfert a été demandé
   :reqjson string sourcePath: *Déprécié*. Le chemin du fichier source 
   :reqjson string destPath: *Déprécié*. Le chemin de destination du fichier 
   :reqjson string file: Le chemin du fichier à transférer
   :reqjson string output: Le chemin de destination du fichier
   :reqjson date start: La date de début du transfert (en format ISO 8601)
   :reqjson object transferInfo: Des informations de transfert personnalisées sous
     la forme d'une liste de pairs clé:valeur, c'est-à-dire sous forme d'un objet JSON.

   :statuscode 202: Le transfert a été lancé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du transfert sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau transfert créé


   **Exemple de requête**

   .. code-block:: http

      POST https://my_waarp_gateway.net/api/transfers HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 212

      {
        "isServer": false,
        "rule": "règle_1",
        "account": "toto",
        "partner": "waarp_sftp",
        "file": "chemin/du/fichier",
        "output": "destination/du/fichier",
        "start": "2019-01-01T02:00:00+02:00",
        "transferInfo": { "key1": "val1", "key2": 2, "key3": true }
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 202 ACCEPTED
      Location: https://my_waarp_gateway.net/api/transfers/123
