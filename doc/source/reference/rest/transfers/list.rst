Lister les transferts
=====================

.. _RFC 3339: https://www.ietf.org/rfc/rfc3339.txt

.. http:get:: /api/transfers

   Renvoie une liste des transferts remplissant les critères donnés en
   paramètre de requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les transferts seront triés.
      Valeurs possibles : ``start+``, ``start-``, ``id+``, ``id-``, ``status+``,
      ``status-``.
      *(défaut: start+)*
   :type sort: string
   :param remote: Filtre uniquement les transferts avec le partenaire renseigné.
      Peut être renseigné plusieurs fois pour filtrer plusieurs partenaires.
   :type remote: int
   :param account: Filtre uniquement les transferts avec le compte renseigné.
      Peut être renseigné plusieurs fois pour filtrer plusieurs comptes.
   :type account: int
   :param rule: Filtre uniquement les transferts avec la règle renseignée.
      Peut être renseigné plusieurs fois pour filtrer plusieurs règles.
   :type rule: int
   :param status: Filtre uniquement les transferts ayant le statut renseigné.
      Valeurs possibles : ``PLANNED``, ``RUNNING``, ``PAUSED``.
      Peut être renseigné plusieurs fois pour filtrer plusieurs status.
   :type status: string
   :param start: Filtre uniquement les transferts dont la date est ultérieure à
      celle renseignée.
   :type start: date

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array transfers: La liste des transferts demandés
   :resjsonarr number id: L'identifiant unique du transfert
   :resjsonarr bool isServer: Précise si la gateway était à l'origine du transfert
   :resjsonarr string rule: L'identifiant de la règle de transfert
   :resjsonarr string requester: Le nom du compte ayant demandé le transfert
   :resjsonarr string requested: Le nom du serveur/partenaire auquel le transfert a été demandé
   :resjsonarr string trueFilepath: Le chemin local complet du fichier (OBSOLÈTE: remplacé par 'localPath')
   :resjsonarr string sourcePath: Le fichier source du transfer (OBSOLÈTE: remplacé par 'localPath' & 'remotePath')
   :resjsonarr string destPath: Le fichier destination du transfer (OBSOLÈTE: remplacé par 'localPath' & 'remotePath')
   :resjsonarr string localPath: Le chemin du fichier sur le disque local
   :resjsonarr string remotePath: Le chemin du fichier sur le partenaire distant
   :resjsonarr number filesize: La taille du fichier (-1 si inconnue)
   :resjsonarr date start: La date de début du transfert
   :resjsonarr string status: Le statut actuel du transfert (*PLANNED*, *RUNNING*, *PAUSED* ou *INTERRUPTED*)
   :resjsonarr string step: L'étape actuelle du transfert (*NONE*, *PRE TASKS*, *DATA*, *POST TASKS*, *ERROR TASKS* ou *FINALIZATION*)
   :resjsonarr number progress: La progression (en octets) du transfert de données
   :resjsonarr number taskNumber: Le numéro du traitement en cours d'exécution
   :resjsonarr string errorCode: Le code d'erreur du transfert (si une erreur s'est produite)
   :resjsonarr string errorMsg: Le message d'erreur du transfert (si une erreur s'est produite)


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/transfers?limit=10&order=desc&rule=1&start=2019-01-01T01:00:00+02:00 HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 249

         {
           "transfers": [{
             "id": 1,
             "isServer": false,
             "rule": "règle_1",
             "requester": "toto",
             "requested": "waarp_sftp",
             "localPath": "/chemin/local/fichier1",
             "remotePath": "/chemin/distant/fichier1",
             "start": "2019-01-01T02:00:00+02:00",
             "status": "RUNNING",
             "step": "DATA",
             "progress": 123456,
           },{
             "id": 2,
             "isServer": true,
             "rule": "règle_2",
             "requester": "tata",
             "requested": "sftp_serveur",
             "localPath": "/chemin/local/fichier2",
             "remotePath": "/chemin/distant/fichier2",
             "start": "2019-01-01T03:00:00+02:00",
             "status": "PLANNED"
           }]
         }
