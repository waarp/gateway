Programmer un transfert
=======================

.. http:post:: /api/transfers

   Programme un nouveau transfert avec les informations renseignées en format JSON dans
   le corps de la requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson bool isServer: Précise si la gateway était à l'origine du transfert
   :reqjson string rule: L'identifiant de la règle utilisée
   :reqjson string requester: Le nom du compte ayant demandé le transfert
   :reqjson string requested: Le nom du serveur/partenaire auquel le transfert a été demandé
   :reqjson string sourcePath: Le chemin du fichier source (OBSOLÈTE: remplacé par 'file')
   :reqjson string destPath: Le chemin de destination du fichier (OBSOLÈTE: remplacé par 'file')
   :reqjson string file: Le nom du fichier à transférer
   :reqjson date start: La date de début du transfert (en format ISO 8601)

   :statuscode 202: Le transfert a été lancé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du transfert sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau transfert créé


   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/transfers HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 212

         {
           "isServer": false,
           "rule": "règle_1",
           "requester": "toto",
           "requested": "waarp_sftp",
           "sourcePath": "chemin/du/fichier",
           "destPath": "chemin/de/destination",
           "start": "2019-01-01T02:00:00+02:00"
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 202 ACCEPTED
         Location: https://my_waarp_gateway.net/api/transfers/123
