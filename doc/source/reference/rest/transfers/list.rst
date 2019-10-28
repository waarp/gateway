Lister les transferts
=====================

.. _RFC 3339: https://www.ietf.org/rfc/rfc3339.txt

.. http:get:: /api/transfers

   Renvoie une liste des transferts remplissant les critères donnés en
   paramètre de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les transferts seront triés *(défaut: start)*
   :type sortby: [start|id|status|rule_id]
   :param order: L'ordre dans lequel les serveurs sont triés *(défaut: asc)*
   :type order: [asc|desc]
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
      Peut être renseigné plusieurs fois pour filtrer plusieurs status.
   :type status: [PLANNED|TRANSFER]
   :param start: Filtre uniquement les transferts dont la date est ultérieure à
      celle renseignée. La date doit être renseignée en format ISO 8601 tel qu'il
      est spécifié dans la `RFC 3339`_.
   :type start: date

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/transfers?limit=10&order=desc&rule=1&start=2019-01-01T01:00:00+02:00 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array transfers: La liste des transferts demandés
   :resjsonarr number id: L'identifiant unique du transfert
   :resjsonarr number ruleID: L'identifiant de la règle de transfert
   :resjsonarr number remoteID: L'identifiant du partenaire de transfert
   :resjsonarr number accountID: L'identifiant du compte de transfert
   :resjsonarr string source: Le chemin d'origine du fichier
   :resjsonarr string destination: Le chemin de destination du fichier
   :resjsonarr date start: La date de début du transfert
   :resjsonarr string status: Le statut actuel du transfert (*PLANNED* ou *TRANSFER*)

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 249

          {
            "transfers": [{
              "id": 1,
              "ruleID": 1,
              "remoteID": 1,
              "accountID": 1,
              "source": "chemin/source/fichier1",
              "destination": "chemin/dest/fichier1",
              "start": "2019-01-01T02:00:00+02:00",
              "status": "TRANSFER"
            },{
              "id": 2,
              "ruleID": 1,
              "remoteID": 2,
              "accountID": 2,
              "source": "chemin/source/fichier2",
              "destination": "chemin/dest/fichier2",
              "start": "2019-01-01T03:00:00+02:00",
              "status": "PLANNED",
            }]
          }